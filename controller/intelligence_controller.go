package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/mail"
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
	analyzerURL   string
	internalToken string
}

func NewIntelligenceController(githubService *service.GithubService, db database.DBTX, analyzerURL, internalToken string) *IntelligenceController {
	return &IntelligenceController{
		githubService: githubService,
		queries:       database.New(db),
		analyzerURL:   strings.TrimSuffix(analyzerURL, "/"),
		internalToken: internalToken,
	}
}

// getAnalyzerURL returns the base URL of the Talent Analyzer service.
func (c *IntelligenceController) getAnalyzerURL() string {
	return c.analyzerURL
}

// getInternalToken returns the shared secret for internal service auth.
func (c *IntelligenceController) getInternalToken() string {
	return c.internalToken
}

// newAnalyzerRequest creates an HTTP request to the Analyzer with the
// internal service token header attached.
func (c *IntelligenceController) newAnalyzerRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token := c.getInternalToken(); token != "" {
		req.Header.Set("X-Internal-Service-Token", token)
	}
	return req, nil
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

type IntelligenceQuizAnswer struct {
	ID               pgtype.UUID      `json:"id"`
	QuizAttemptID    pgtype.UUID      `json:"quiz_attempt_id"`
	QuestionID       pgtype.UUID      `json:"question_id"`
	UserAnswer       pgtype.Text      `json:"user_answer"`
	IsCorrect        pgtype.Bool      `json:"is_correct"`
	LastSavedAt      pgtype.Timestamp `json:"last_saved_at"`
	SaveCount        pgtype.Int4      `json:"save_count"`
	TimeSpentSeconds pgtype.Int4      `json:"time_spent_seconds"`
	ExecutionTimeMs  pgtype.Int4      `json:"execution_time_ms"`
	MemoryUsedMb     pgtype.Float8    `json:"memory_used_mb"`
	IsSkipped        pgtype.Bool      `json:"is_skipped"`
	IsReviewed       pgtype.Bool      `json:"is_reviewed"`
	CreatedAt        pgtype.Timestamp `json:"created_at"`
	UpdatedAt        pgtype.Timestamp `json:"updated_at"`
}

type IntelligenceReport struct {
	UserID      uuid.UUID                `json:"user_id"`
	GitHub      GitHubIntelligence       `json:"github_intelligence"`
	CVSignals   *CVSignalsResponse       `json:"cv_signals,omitempty"`
	AISummary   *AISummaryResponse       `json:"ai_summary,omitempty"`
	QuizAnswers []IntelligenceQuizAnswer `json:"quiz_answers,omitempty"`
}

// --- Handler ---

func (c *IntelligenceController) FetchGitHubSnapshot(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	// Only the user themselves or an admin can fetch the GitHub intelligence snapshot
	if claims.Role != "admin" && claims.UserID != targetUserID {
		writeError(w, http.StatusForbidden, "Access denied: you can only fetch your own GitHub intelligence")
		return
	}

	var pgID pgtype.UUID
	copy(pgID.Bytes[:], targetUserID[:])
	pgID.Valid = true

	// 1. Check for existing cached snapshot (one current snapshot per user, 24h freshness)
	var githubUser *service.GitHubUser
	var repos []service.GitHubRepo

	snapshot, err := c.queries.GetLatestGitHubSnapshot(r.Context(), targetUserID)
	if err == nil && snapshot.UpdatedAt.Valid && time.Since(snapshot.UpdatedAt.Time) < 24*time.Hour && len(snapshot.RawData) > 0 {
		var cached struct {
			User  *service.GitHubUser  `json:"user"`
			Repos []service.GitHubRepo `json:"repos"`
		}
		if unmarshalErr := json.Unmarshal(snapshot.RawData, &cached); unmarshalErr == nil && cached.User != nil {
			githubUser = cached.User
			repos = cached.Repos
		}
	}

	// 2. Fetch from GitHub API and upsert single user snapshot if cache missed or expired
	if githubUser == nil {
		dbUser, err := c.queries.GetUserByID(r.Context(), pgID)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		githubUser, repos, err = c.githubService.GetUserWithDetails(r.Context(), dbUser.GithubAccessToken.String)
		if err != nil {
			http.Error(w, "Failed to fetch from GitHub: "+err.Error(), http.StatusInternalServerError)
			return
		}

		rawData := map[string]interface{}{"user": githubUser, "repos": repos}
		rawBytes, _ := json.Marshal(rawData)

		_, err = c.queries.CreateGitHubSnapshot(r.Context(), database.CreateGitHubSnapshotParams{
			UserID:      targetUserID,
			PublicRepos: pgtype.Int4{Int32: int32(githubUser.PublicRepos), Valid: true},
			Followers:   pgtype.Int4{Int32: int32(githubUser.Followers), Valid: true},
			Following:   pgtype.Int4{Int32: int32(githubUser.Following), Valid: true},
			RawData:     rawBytes,
		})
		if err != nil {
			http.Error(w, "Failed to save snapshot: "+err.Error(), http.StatusInternalServerError)
			return
		}
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
	aiSummary, err := c.queries.GetLatestAISummary(r.Context(), targetUserID)
	if err == nil {
		report.AISummary = &AISummaryResponse{
			Summary:    string(aiSummary.Summary),
			Strengths:  aiSummary.Strengths.String,
			Weaknesses: aiSummary.Weaknesses.String,
			Model:      aiSummary.Model.String,
		}
	}

	// 7. Attach quiz answers if available (excluding code_output)
	quizAnswers, err := c.queries.GetUserQuizAnswers(r.Context(), pgID)
	if err == nil {
		cleanAnswers := make([]IntelligenceQuizAnswer, len(quizAnswers))
		for i, a := range quizAnswers {
			cleanAnswers[i] = IntelligenceQuizAnswer{
				ID:               a.ID,
				QuizAttemptID:    a.QuizAttemptID,
				QuestionID:       a.QuestionID,
				UserAnswer:       a.UserAnswer,
				IsCorrect:        a.IsCorrect,
				LastSavedAt:      a.LastSavedAt,
				SaveCount:        a.SaveCount,
				TimeSpentSeconds: a.TimeSpentSeconds,
				ExecutionTimeMs:  a.ExecutionTimeMs,
				MemoryUsedMb:     a.MemoryUsedMb,
				IsSkipped:        a.IsSkipped,
				IsReviewed:       a.IsReviewed,
				CreatedAt:        a.CreatedAt,
				UpdatedAt:        a.UpdatedAt,
			}
		}
		report.QuizAnswers = cleanAnswers
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (c *IntelligenceController) GetLatestUserSummary(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok || claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	if c.queries == nil {
		writeError(w, http.StatusInternalServerError, "AI summary service is not configured")
		return
	}

	summary, err := c.queries.GetLatestAISummary(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "AI summary not found for user")
		return
	}

	resp := map[string]interface{}{
		"id":         summary.ID,
		"user_id":    summary.UserID,
		"summary":    json.RawMessage(summary.Summary),
		"strengths":  summary.Strengths.String,
		"weaknesses": summary.Weaknesses.String,
		"model":      summary.Model.String,
		"created_at": summary.CreatedAt,
	}
	if summary.CvVersion.Valid {
		resp["cv_version"] = int(summary.CvVersion.Int32)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (c *IntelligenceController) AnalyzeCV(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file field. Use form field name 'file'.")
		return
	}
	defer file.Close()

	githubUsername := r.FormValue("github_username")
	cvVersionStr := r.FormValue("cv_version")
	cvVersion, _ := strconv.Atoi(cvVersionStr)

	var targetUserID uuid.UUID
	if githubUsername != "" {
		if u, err := c.queries.GetUserByGitHubUsername(r.Context(), githubUsername); err == nil {
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

		existingSummary, err := c.queries.GetAISummaryByCVVersion(r.Context(), database.GetAISummaryByCVVersionParams{
			UserID:    pgUserID.Bytes,
			CvVersion: pgVersion,
		})

		if err == nil {
			// Found it! Return the existing summary and skip calling the talent-analyzer
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(existingSummary.Summary)
			return
		}
	}
	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	fw, err := mp.CreateFormFile("file", filepath.Base(header.Filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create multipart writer")
		return
	}
	if _, err := io.Copy(fw, file); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to copy file data")
		return
	}

	if err := mp.WriteField("github_username", githubUsername); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write form field")
		return
	}
	mp.Close()

	analyzerURL := c.getAnalyzerURL() + "/analyze-cv"

	client := &http.Client{Timeout: 300 * time.Second}
	req, err := c.newAnalyzerRequest(http.MethodPost, analyzerURL, &buf)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	req.Header.Set("Content-Type", mp.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to reach analysis service: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read analysis response")
		return
	}

	// Save to history file
	historyDir := "history"
	os.MkdirAll(historyDir, 0755)
	stem := filepath.Base(header.Filename)
	if ext := filepath.Ext(stem); ext != "" {
		stem = stem[:len(stem)-len(ext)]
	}
	ts := time.Now().Unix()
	historyPath := filepath.Join(historyDir, fmt.Sprintf("analyze_%s_%d.json", stem, ts))
	os.WriteFile(historyPath, body, 0644)

	// Store the full analyzer payload in the database.
	var analysisResp struct {
		Analysis json.RawMessage `json:"analysis"`
		Response json.RawMessage `json:"response"`
		Engine   string          `json:"engine"`
	}
	if err := json.Unmarshal(body, &analysisResp); err == nil {
		var userID uuid.UUID
		if githubUsername != "" {
			if u, err := c.queries.GetUserByGitHubUsername(r.Context(), githubUsername); err == nil {
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

			strengths, weaknesses := extractStrengthsWeaknesses(analysisPayload)
			_, _ = c.queries.CreateAISummary(r.Context(), database.CreateAISummaryParams{
				UserID:     userID,
				Summary:    body,
				Strengths:  pgtype.Text{String: strengths, Valid: strengths != ""},
				Weaknesses: pgtype.Text{String: weaknesses, Valid: weaknesses != ""},
				Model:      pgtype.Text{String: analysisResp.Engine, Valid: analysisResp.Engine != ""},
				CvVersion:  pgtype.Int4{Int32: int32(cvVersion), Valid: cvVersion > 0},
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (c *IntelligenceController) GenerateJobDescription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt      string `json:"prompt"`
		CompanyName string `json:"company_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	payload, _ := json.Marshal(req)

	url := c.getAnalyzerURL() + "/generate-job-description"

	client := &http.Client{Timeout: 120 * time.Second}
	httpReq, err := c.newAnalyzerRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to reach AI service: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read AI response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

type ContactRequest struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Company        string `json:"company"`
	BudgetRange    string `json:"budget_range"`
	ProjectDetails string `json:"project_details"`
}

func (c *IntelligenceController) Contact(w http.ResponseWriter, r *http.Request) {
	var req ContactRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	req.Company = strings.TrimSpace(req.Company)
	req.BudgetRange = strings.TrimSpace(req.BudgetRange)
	req.ProjectDetails = strings.TrimSpace(req.ProjectDetails)
	if req.FirstName == "" || req.LastName == "" || req.Email == "" || req.ProjectDetails == "" {
		writeError(w, http.StatusBadRequest, "Required contact fields are missing")
		return
	}
	if len(req.FirstName) > 100 || len(req.LastName) > 100 || len(req.Email) > 255 || len(req.Company) > 255 || len(req.BudgetRange) > 100 || len(req.ProjectDetails) > 10000 {
		writeError(w, http.StatusBadRequest, "Contact request fields are too long")
		return
	}
	parsedEmail, err := mail.ParseAddress(req.Email)
	if err != nil || parsedEmail.Address != req.Email {
		writeError(w, http.StatusBadRequest, "Invalid email address")
		return
	}

	company := pgtype.Text{String: req.Company, Valid: req.Company != ""}
	budgetRange := pgtype.Text{String: req.BudgetRange, Valid: req.BudgetRange != ""}
	if _, err := c.queries.CreateContactRequest(r.Context(), database.CreateContactRequestParams{
		FirstName: req.FirstName, LastName: req.LastName, Email: req.Email,
		Company: company, BudgetRange: budgetRange, ProjectDetails: req.ProjectDetails,
	}); err != nil {
		log.Printf("ERROR: failed to store contact request: %v", err)
		writeError(w, http.StatusInternalServerError, "Unable to process contact request")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message":"Contact inquiry sent successfully"}`))
}

func (c *IntelligenceController) GenerateJobDescriptionPublic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt      string `json:"prompt"`
		CompanyName string `json:"company_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	payload, _ := json.Marshal(req)

	url := c.getAnalyzerURL() + "/generate-job-description"

	client := &http.Client{Timeout: 120 * time.Second}
	httpReq, err := c.newAnalyzerRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to reach AI service: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read AI response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (c *IntelligenceController) GenerateQuestions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt       string `json:"prompt"`
		QuestionType string `json:"question_type,omitempty"`
		Difficulty   string `json:"difficulty,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	payload, _ := json.Marshal(req)

	url := c.getAnalyzerURL() + "/generate-questions"

	client := &http.Client{Timeout: 120 * time.Second}
	httpReq, err := c.newAnalyzerRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to reach AI service: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read AI response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (c *IntelligenceController) GenerateQuestionsPublic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt       string `json:"prompt"`
		QuestionType string `json:"question_type,omitempty"`
		Difficulty   string `json:"difficulty,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	payload, _ := json.Marshal(req)

	url := c.getAnalyzerURL() + "/generate-questions"

	client := &http.Client{Timeout: 120 * time.Second}
	httpReq, err := c.newAnalyzerRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to reach AI service: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read AI response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func extractStrengthsWeaknesses(response json.RawMessage) (string, string) {
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
	var strengths, weaknesses []string
	for _, c := range data.Checks {
		if c.Status == "pass" {
			strengths = append(strengths, c.Message)
		} else if c.Status == "warn" || c.Status == "fail" {
			weaknesses = append(weaknesses, c.Message)
		}
	}
	return strings.Join(strengths, "; "), strings.Join(weaknesses, "; ")
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
		"Go": true, "Python": true, "Java": true, "Rust": true,
		"C": true, "C++": true, "C#": true, "Ruby": true, "PHP": true,
		"Kotlin": true, "Swift": true, "Elixir": true, "Scala": true,
	}
	frontendLangs := map[string]bool{
		"JavaScript": true, "TypeScript": true, "HTML": true,
		"CSS": true, "Vue": true, "Svelte": true, "Dart": true,
	}
	devopsLangs := map[string]bool{
		"Shell": true, "Dockerfile": true, "HCL": true, "Makefile": true,
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
