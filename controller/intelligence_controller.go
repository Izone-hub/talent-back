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
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const analyzeCVURL = "http://localhost:8000/analyze-cv"

type IntelligenceController struct {
	githubService *service.GithubService
	queries       *database.Queries
}

func NewIntelligenceController(githubService *service.GithubService, db database.DBTX) *IntelligenceController {
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
	UserID    uuid.UUID          `json:"user_id"`
	GitHub    GitHubIntelligence `json:"github_intelligence"`
	CVSignals *CVSignalsResponse `json:"cv_signals,omitempty"`
	AISummary *AISummaryResponse `json:"ai_summary,omitempty"`
}

// --- Handler ---

func (c *IntelligenceController) FetchGitHubSnapshot(w http.ResponseWriter, r *http.Request) {
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
	githubUser, repos, err := c.githubService.GetUserWithDetails(r.Context(), dbUser.GithubAccessToken.String)
	if err != nil {
		http.Error(w, "Failed to fetch from GitHub: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Serialize raw data and save snapshot
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
			Summary:    aiSummary.Summary.String,
			Strengths:  aiSummary.Strengths.String,
			Weaknesses: aiSummary.Weaknesses.String,
			Model:      aiSummary.Model.String,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
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

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post(analyzeCVURL, mp.FormDataContentType(), &buf)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
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
