package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type IntelligenceController struct {
	githubService *service.GithubService
	queries       *database.Queries
}

func NewIntelligenceController(
	githubService *service.GithubService,
	db database.DBTX,
) *IntelligenceController {
	return &IntelligenceController{
		githubService: githubService,
		queries:       database.New(db),
	}
}

// --- Response types ---

type GitHubIntelligence struct {
	ActivityLevel string   `json:"activity_level"` // low / min / mid / max
	Focus         string   `json:"focus"`          // Backend / Frontend / Fullstack / DevOps / Mixed
	TopLanguages  []string `json:"top_languages"`
	PublicRepos   int      `json:"public_repos"`
	Followers     int      `json:"followers"`
	Following     int      `json:"following"`
}

type CVSignalsResponse struct {
	ClaimedSkills       []string `json:"claimed_skills"`
	ExperienceLevel     string   `json:"experience_level"`
	ProjectsListed      int      `json:"projects_listed"`
	Credibility         string   `json:"credibility"`
	AlignmentWithGitHub string   `json:"alignment_with_github"`
}

type AISummaryResponse struct {
	Summary    string `json:"summary"`
	Strengths  string `json:"strengths"`
	Weaknesses string `json:"weaknesses"`
	Model      string `json:"model"`
}

type IntelligenceReport struct {
	UserID      uuid.UUID              `json:"user_id"`
	GitHub      GitHubIntelligence     `json:"github_intelligence"`
	CVSignals   *CVSignalsResponse     `json:"cv_signals,omitempty"`
	AISummary   *AISummaryResponse     `json:"ai_summary,omitempty"`
	QuizAnswers []database.QuizAnswer  `json:"quiz_answers,omitempty"`
}

// --- Handler ---

func (c *IntelligenceController) FetchGitHubSnapshot(
	w http.ResponseWriter,
	r *http.Request,
) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var pgID pgtype.UUID
	copy(pgID.Bytes[:], targetUserID[:])
	pgID.Valid = true

	// 1. Fetch user to get GithubAccessToken
	dbUser, err := c.queries.GetUserByID(r.Context(), pgID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 2. Call GitHub API
	githubUser, repos, err := c.githubService.GetUserWithDetails(
		r.Context(),
		dbUser.GithubAccessToken.String,
	)
	if err != nil {
		http.Error(
			w,
			"Failed to fetch from GitHub: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// 3. Serialize raw data and save snapshot
	rawData := map[string]interface{}{
		"user":  githubUser,
		"repos": repos,
	}

	rawBytes, _ := json.Marshal(rawData)

	_, err = c.queries.CreateGitHubSnapshot(
		r.Context(),
		database.CreateGitHubSnapshotParams{
			UserID:      targetUserID,
			PublicRepos: pgtype.Int4{Int32: int32(githubUser.PublicRepos), Valid: true},
			Followers:   pgtype.Int4{Int32: int32(githubUser.Followers), Valid: true},
			Following:   pgtype.Int4{Int32: int32(githubUser.Following), Valid: true},
			RawData:     rawBytes,
		},
	)
	if err != nil {
		http.Error(
			w,
			"Failed to save snapshot: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// 4. Compute GitHub Intelligence signals
	topLanguages := c.githubService.CalculateTopLanguages(repos)
	activityLevel := computeActivityLevel(githubUser.PublicRepos)
	focus := computeFocus(topLanguages)

	report := IntelligenceReport{
		UserID: targetUserID,
		GitHub: GitHubIntelligence{
			ActivityLevel: activityLevel,
			Focus:         focus,
			TopLanguages:  topLanguages,
			PublicRepos:   githubUser.PublicRepos,
			Followers:     githubUser.Followers,
			Following:     githubUser.Following,
		},
	}

	// 5. Attach CV signals if available
	cvSignals, err := c.queries.GetCVSignalsByUser(r.Context(), pgID)
	if err == nil {
		var skills []string
		_ = json.Unmarshal(cvSignals.ClaimedSkills, &skills)

		report.CVSignals = &CVSignalsResponse{
			ClaimedSkills:       skills,
			ExperienceLevel:     cvSignals.ExperienceLevel.String,
			ProjectsListed:      int(cvSignals.ProjectsListed.Int32),
			Credibility:         cvSignals.Credibility.String,
			AlignmentWithGitHub: cvSignals.AlignmentWithGithub.String,
		}
	}

	// 6. Attach latest AI summary if available
	aiSummary, err := c.queries.GetLatestAISummary(
		r.Context(),
		targetUserID,
	)
	if err == nil {
		report.AISummary = &AISummaryResponse{
			Summary:    string(aiSummary.Summary),
			Strengths:  aiSummary.Strengths.String,
			Weaknesses: aiSummary.Weaknesses.String,
			Model:      aiSummary.Model.String,
		}
	}

	// 7. Attach quiz answers if available
	quizAnswers, err := c.queries.GetUserQuizAnswers(
		r.Context(),
		pgID,
	)
	if err == nil {
		report.QuizAnswers = quizAnswers
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (c *IntelligenceController) AnalyzeCV(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"Failed to parse form: "+err.Error(),
		)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"Missing file field. Use form field name 'file'.",
		)
		return
	}
	defer file.Close()

	githubUsername := r.FormValue("github_username")
	cvVersionStr := r.FormValue("cv_version")
	cvVersion, _ := strconv.Atoi(cvVersionStr)

	var targetUserID uuid.UUID

	if githubUsername != "" {
		if u, err := c.queries.GetUserByGitHubUsername(
			r.Context(),
			githubUsername,
		); err == nil {
			targetUserID = uuid.UUID(u.ID.Bytes)
		}
	}

	if targetUserID != uuid.Nil && cvVersion > 0 {
		var pgUserID pgtype.UUID
		copy(pgUserID.Bytes[:], targetUserID[:])
		pgUserID.Valid = true

		var pgVersion pgtype.Int4
		pgVersion.Int32 = int32(cvVersion)
		pgVersion.Valid = true

		// Check if an AI summary already exists for this CV version.
		existingSummary, err := c.queries.GetAISummaryByCVVersion(
			r.Context(),
			database.GetAISummaryByCVVersionParams{
				UserID:    pgUserID.Bytes,
				CvVersion: pgVersion,
			},
		)

		if err == nil {
			// Found it. Return the existing summary and skip
			// calling the talent-analyzer.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(existingSummary.Summary)
			return
		}
	}

	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	fw, err := mp.CreateFormFile(
		"file",
		filepath.Base(header.Filename),
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to create multipart writer",
		)
		return
	}

	if _, err := io.Copy(fw, file); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to copy file data",
		)
		return
	}

	if err := mp.WriteField("github_username", githubUsername); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to write form field",
		)
		return
	}

	if err := mp.Close(); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to close multipart writer",
		)
		return
	}

	analyzerURL := os.Getenv("ANALYZER_URL")
	if analyzerURL == "" {
		analyzerURL = "http://localhost:8000/analyze-cv"
	}

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	resp, err := client.Post(
		analyzerURL,
		mp.FormDataContentType(),
		&buf,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadGateway,
			fmt.Sprintf("Failed to reach analysis service: %v", err),
		)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to read analysis response",
		)
		return
	}

	// Save to history file
	historyDir := "history"
	_ = os.MkdirAll(historyDir, 0755)

	stem := filepath.Base(header.Filename)
	if ext := filepath.Ext(stem); ext != "" {
		stem = stem[:len(stem)-len(ext)]
	}

	ts := time.Now().Unix()
	historyPath := filepath.Join(
		historyDir,
		fmt.Sprintf("analyze_%s_%d.json", stem, ts),
	)

	_ = os.WriteFile(historyPath, body, 0644)

	// Store the full analyzer payload in the database.
	var analysisResp struct {
		Analysis json.RawMessage `json:"analysis"`
		Response json.RawMessage `json:"response"`
		Engine   string          `json:"engine"`
	}

	if err := json.Unmarshal(body, &analysisResp); err == nil {
		var userID uuid.UUID

		if githubUsername != "" {
			if u, err := c.queries.GetUserByGitHubUsername(
				r.Context(),
				githubUsername,
			); err == nil {
				userID = uuid.UUID(u.ID.Bytes)
			}
		}

		if userID != uuid.Nil {
			analysisPayload := analysisResp.Analysis

			if len(analysisPayload) == 0 {
				analysisPayload = analysisResp.Response
			}

			if len(analysisPayload) == 0 {
				analysisPayload = body
			}

			strengths, weaknesses := extractStrengthsWeaknesses(
				analysisPayload,
			)

			_, _ = c.queries.CreateAISummary(
				r.Context(),
				database.CreateAISummaryParams{
					UserID:     userID,
					Summary:    body,
					Strengths:  pgtype.Text{String: strengths, Valid: strengths != ""},
					Weaknesses: pgtype.Text{String: weaknesses, Valid: weaknesses != ""},
					Model:      pgtype.Text{String: analysisResp.Engine, Valid: analysisResp.Engine != ""},
					CvVersion:  pgtype.Int4{Int32: int32(cvVersion), Valid: cvVersion > 0},
				},
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (c *IntelligenceController) GenerateJobDescription(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req struct {
		Prompt      string `json:"prompt"`
		CompanyName string `json:"company_name,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"Invalid request body: "+err.Error(),
		)
		return
	}

	if req.Prompt == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"prompt is required",
		)
		return
	}

	payload, _ := json.Marshal(req)

	analyzerURL := os.Getenv("ANALYZER_URL")
	if analyzerURL == "" {
		analyzerURL = "http://localhost:8000"
	}

	url := strings.TrimSuffix(
		analyzerURL,
		"/analyze-cv",
	) + "/generate-job-description"

	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	resp, err := client.Post(
		url,
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadGateway,
			fmt.Sprintf("Failed to reach AI service: %v", err),
		)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"Failed to read AI response",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func extractStrengthsWeaknesses(
	response json.RawMessage,
) (string, string) {
	var data struct {
		Checks []struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		} `json:"checks"`
	}

	if err := json.Unmarshal(response, &data); err != nil {
		return "", ""
	}

	var strengths []string
	var weaknesses []string

	for _, c := range data.Checks {
		if c.Status == "pass" {
			strengths = append(strengths, c.Message)
		} else if c.Status == "warn" || c.Status == "fail" {
			weaknesses = append(weaknesses, c.Message)
		}
	}

	return strings.Join(strengths, "; "),
		strings.Join(weaknesses, "; ")
}

// --- Helpers ---

func computeActivityLevel(repos int) string {
	switch {
	case repos <= 4:
		return "low"
	case repos <= 15:
		return "min"
	case repos <= 30:
		return "mid"
	default:
		return "max"
	}
}

func computeFocus(languages []string) string {
	backendLangs := map[string]bool{
		"Go": true,
		"Python": true,
		"Java": true,
		"Rust": true,
		"C": true,
		"C++": true,
		"C#": true,
		"Ruby": true,
		"PHP": true,
		"Kotlin": true,
		"Swift": true,
		"Elixir": true,
		"Scala": true,
	}

	frontendLangs := map[string]bool{
		"JavaScript": true,
		"TypeScript": true,
		"HTML": true,
		"CSS": true,
		"Vue": true,
		"Svelte": true,
		"Dart": true,
	}

	devopsLangs := map[string]bool{
		"Shell":      true,
		"Dockerfile": true,
		"HCL":        true,
		"Makefile":   true,
	}

	backendCount, frontendCount, devopsCount := 0, 0, 0

	for _, lang := range languages {
		if backendLangs[lang] {
			backendCount++
		} else if frontendLangs[lang] {
			frontendCount++
		} else if devopsLangs[lang] {
			devopsCount++
		}
	}

	// Determine dominant focus
	if devopsCount > backendCount && devopsCount > frontendCount {
		return "DevOps"
	}

	if backendCount > 0 && frontendCount > 0 {
		return "Fullstack"
	}

	if backendCount > frontendCount {
		return "Backend Systems"
	}

	if frontendCount > backendCount {
		return "Frontend"
	}

	return "Mixed"
}