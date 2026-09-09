package controller

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type ApplicationController struct {
	appService      *service.ApplicationService
	cvService       *service.CvService
	feedbackService *service.QuizResultFeedbackService
	analyzerURL     string
	internalToken   string
}

func NewApplicationController(appService *service.ApplicationService, cvService *service.CvService, feedbackService *service.QuizResultFeedbackService, analyzerURL, internalToken string) *ApplicationController {
	if analyzerURL == "" {
		analyzerURL = "http://localhost:8000"
	}
	return &ApplicationController{
		appService:      appService,
		cvService:       cvService,
		feedbackService: feedbackService,
		analyzerURL:     strings.TrimSuffix(analyzerURL, "/"),
		internalToken:   internalToken,
	}
}

func (c *ApplicationController) ApplyForJob(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	app, err := c.appService.ApplyForJob(r.Context(), jobID, claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if cv, err := c.cvService.GetCurrentCV(r.Context(), claims.UserID); err == nil {
		go triggerCVAnalysis(cv.FilePath, cv.FileName, claims.GithubUsername, cv.Version)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Successfully applied for the job",
		"application_id": app.ID,
	})
}

func (c *ApplicationController) GetMyApplications(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	apps, err := c.appService.GetMyApplications(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}

func (c *ApplicationController) GetJobApplications(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	apps, err := c.appService.GetApplicationsForJob(r.Context(), jobID, 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}

// GetUserApplications lists every application a user has submitted across all
// jobs (admin only). Used by the admin user detail page.
func (c *ApplicationController) GetUserApplications(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	apps, err := c.appService.GetUserApplications(r.Context(), userID, 100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}

func (c *ApplicationController) GetApplicationDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	app, err := c.appService.GetApplicationDetail(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Application not found: "+err.Error())
		return
	}

	appBytes, err := json.Marshal(app)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(appBytes, &resp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Read candidate feedback directly from public.quiz_result_feedback table
	if c.feedbackService != nil {
		if fb, fbErr := c.feedbackService.GetByApplication(r.Context(), appID); fbErr == nil && fb != nil {
			resp["candidate_feedback"] = fb
			resp["CandidateFeedback"] = fb
			resp["candidate_feedback_rating"] = fb.Rating
			resp["CandidateFeedbackRating"] = fb.Rating
			resp["candidate_feedback_comment"] = fb.Comment
			resp["CandidateFeedbackComment"] = fb.Comment
			resp["rating"] = fb.Rating
			resp["Rating"] = fb.Rating
			resp["comment"] = fb.Comment
			resp["Comment"] = fb.Comment
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (c *ApplicationController) GetApplicationInformation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	info, err := c.appService.GetApplicationInformation(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, info)
}

func (c *ApplicationController) StartReview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	app, err := c.appService.StartReview(r.Context(), appID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) ShortlistApplication(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	app, err := c.appService.ShortlistApplication(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) MarkInterviewed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	app, err := c.appService.MarkInterviewed(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) AcceptApplication(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	app, newlyAccepted, err := c.appService.AcceptApplication(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if newlyAccepted {
		go c.sendAcceptanceEmail(app)
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) sendAcceptanceEmail(app database.GetApplicationWithDetailsRow) {
	payload, err := json.Marshal(map[string]interface{}{
		"type":            "job_accepted",
		"recipient_email": app.UserEmail.String,
		"candidate_name":  app.UserName.String,
		"job_title":       app.JobTitle,
		"company_name":    app.JobCompany,
		"job_id":          uuid.UUID(app.JobID.Bytes).String(),
		"accepted_at":     time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		log.Printf("ERROR: failed to encode job acceptance email notification: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, c.analyzerURL+"/api/v1/notifications/job-accepted", bytes.NewReader(payload))
	if err != nil {
		log.Printf("ERROR: failed to create job acceptance email notification: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		req.Header.Set("X-Internal-Service-Token", c.internalToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: failed to send job acceptance email notification: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("ERROR: analyzer rejected job acceptance email notification with status %d", resp.StatusCode)
	}
}

func (c *ApplicationController) RejectApplication(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req struct {
		Reason   string `json:"reason"`
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = ""
		req.Feedback = ""
	}

	app, err := c.appService.RejectApplication(r.Context(), appID, req.Reason, req.Feedback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) WithdrawApplication(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	app, err := c.appService.WithdrawApplication(r.Context(), appID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) ListApplicationsByStatus(w http.ResponseWriter, r *http.Request) {
	statusStr := r.PathValue("status")
	status := database.ApplicationStatus(statusStr)

	apps, err := c.appService.ListApplicationsByStatus(r.Context(), status, 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}

func (c *ApplicationController) GetApplicationCountsByJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	counts, err := c.appService.GetApplicationCountsByJob(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, counts)
}

func (c *ApplicationController) AddEmployerFeedback(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	app, err := c.appService.AddEmployerFeedback(r.Context(), appID, req.Feedback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func (c *ApplicationController) GetRecentApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := c.appService.GetRecentApplications(r.Context(), 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}
