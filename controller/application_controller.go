package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type ApplicationController struct {
	appService *service.ApplicationService
}

func NewApplicationController(appService *service.ApplicationService) *ApplicationController {
	return &ApplicationController{
		appService: appService,
	}
}

// ApplyForJob handles POST /api/v1/jobs/{id}/apply
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Successfully applied for the job",
		"application_id": app.ID,
	})
}

// GetMyApplications handles GET /api/v1/applications/my
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

// GetJobApplications handles GET /api/v1/jobs/{id}/applications (Employer/Admin only)
func (c *ApplicationController) GetJobApplications(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Assuming default pagination
	apps, err := c.appService.GetApplicationsForJob(r.Context(), jobID, 50, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, apps)
}

// AcceptApplication handles PATCH /api/v1/applications/{id}/accept (Employer/Admin only)
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
