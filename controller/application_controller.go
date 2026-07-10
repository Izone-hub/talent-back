package controller

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/service"
)

type ApplicationController struct {
	appService *service.ApplicationService
	cvService  *service.CvService
}

func NewApplicationController(appService *service.ApplicationService, cvService *service.CvService) *ApplicationController {
	return &ApplicationController{
		appService: appService,
		cvService:  cvService,
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
		go triggerCVAnalysis(cv.FilePath, cv.FileName, claims.GithubUsername)
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

	writeJSON(w, http.StatusOK, app)
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

	app, err := c.appService.AcceptApplication(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, app)
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
