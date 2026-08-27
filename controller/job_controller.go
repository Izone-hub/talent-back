package controller

import (
	"encoding/json"
	"net/http"
	"strconv" // string conversion for pagination params

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

// JobController handles HTTP requests for job management.
type JobController struct {
	jobService *service.JobService // dependency injection of service layer
}

// NewJobController creates a new JobController.
func NewJobController(jobService *service.JobService) *JobController {
	return &JobController{
		jobService: jobService,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getUserID extracts the authenticated user's UUID from the request context.
// The auth middleware stores *service.Claims under key "user".
func getUserID(r *http.Request) (uuid.UUID, bool) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		return uuid.Nil, false
	}
	return claims.UserID, true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// parseJobID extracts the job UUID from the URL path.
// Expects the pattern /api/jobs/{id}/... where {id} is captured by
// Go 1.22+ ServeMux path value.
func parseJobID(r *http.Request) (uuid.UUID, error) {
	idStr := r.PathValue("id")
	return uuid.Parse(idStr)
}

// parsePagination reads "limit" and "offset" query params with defaults.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20 // default
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return
}

// ---------------------------------------------------------------------------
// POST /api/jobs — Create a new job (draft)
// ---------------------------------------------------------------------------

func (c *JobController) CreateJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	job, err := c.jobService.CreateJob(r.Context(), userID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, job.ToResponse())
}

// ---------------------------------------------------------------------------
// GET /api/jobs — List published jobs (public)
// ---------------------------------------------------------------------------

func (c *JobController) ListPublishedJobs(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	category := r.URL.Query().Get("category")

	var userIDPtr *uuid.UUID
	if userID, ok := getUserID(r); ok {
		userIDPtr = &userID
	}

	jobs, err := c.jobService.ListPublishedJobs(r.Context(), userIDPtr, category, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response list
	responses := make([]models.JobResponse, 0, len(jobs))
	for _, j := range jobs {
		responses = append(responses, j.ToResponse())
	}

	writeJSON(w, http.StatusOK, models.JobListResponse{
		Jobs:   responses,
		Total:  len(responses),
		Limit:  limit,
		Offset: offset,
	})
}

// ---------------------------------------------------------------------------
// GET /api/jobs/my — List the authenticated user's jobs (any status)
// ---------------------------------------------------------------------------

func (c *JobController) ListMyJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, offset := parsePagination(r)
	category := r.URL.Query().Get("category")

	jobs, err := c.jobService.ListMyJobs(r.Context(), userID, category, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]models.JobResponse, 0, len(jobs))
	for _, j := range jobs {
		responses = append(responses, j.ToResponse())
	}

	writeJSON(w, http.StatusOK, models.JobListResponse{
		Jobs:   responses,
		Total:  len(responses),
		Limit:  limit,
		Offset: offset,
	})
}

// ---------------------------------------------------------------------------
// GET /api/jobs/{id} — Get a single published job (public)
// ---------------------------------------------------------------------------

func (c *JobController) GetPublishedJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := c.jobService.GetPublishedJob(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// ---------------------------------------------------------------------------
// PUT /api/jobs/{id} — Update a draft job (owner only)
// ---------------------------------------------------------------------------

func (c *JobController) UpdateJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	var req models.UpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	job, err := c.jobService.UpdateJob(r.Context(), userID, jobID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// ---------------------------------------------------------------------------
// PATCH /api/jobs/{id}/publish — Draft → Published
// ---------------------------------------------------------------------------

func (c *JobController) PublishJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := c.jobService.PublishJob(r.Context(), userID, jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// ---------------------------------------------------------------------------
// PATCH /api/jobs/{id}/close — Published → Closed
// ---------------------------------------------------------------------------

func (c *JobController) CloseJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := c.jobService.CloseJob(r.Context(), userID, jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// ---------------------------------------------------------------------------
// PATCH /api/jobs/{id}/archive — Published/Closed → Archived
// ---------------------------------------------------------------------------

func (c *JobController) ArchiveJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := c.jobService.ArchiveJob(r.Context(), userID, jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, job.ToResponse())
}

// POST /api/v1/jobs/{id}/save — Save/bookmark a job for the authenticated user
func (c *JobController) SaveJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	var req struct{
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := c.jobService.SaveJob(r.Context(), userID, jobID, req.Notes); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Job saved"})
}

// DELETE /api/v1/jobs/{id}/save — Unsave/remove bookmark
func (c *JobController) UnsaveJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	if err := c.jobService.UnsaveJob(r.Context(), userID, jobID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to unsave job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Job unsaved"})
}

// GET /api/v1/jobs/saved — List the authenticated user's saved/bookmarked jobs
func (c *JobController) ListSavedJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, offset := parsePagination(r)

	savedJobs, total, err := c.jobService.ListSavedJobs(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch saved jobs: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"saved_jobs": savedJobs,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// GET /api/v1/jobs/{id}/saved — Check if the authenticated user saved a job
func (c *JobController) IsJobSaved(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	isSaved, err := c.jobService.IsJobSaved(r.Context(), userID, jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check saved status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"is_saved": isSaved})
}
