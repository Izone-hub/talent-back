package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SavedJobController struct {
	queries *database.Queries
}

func NewSavedJobController(db database.DBTX) *SavedJobController {
	return &SavedJobController{
		queries: database.New(db),
	}
}

// POST /api/v1/jobs/{id}/save — Save/bookmark a job
func (c *SavedJobController) SaveJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	// Notes are optional, so ignore decode errors
	_ = json.NewDecoder(r.Body).Decode(&req)

	pgUserID := uuidToPgtype(userID)
	pgJobID := uuidToPgtype(jobID)

	err = c.queries.SaveJob(r.Context(), database.SaveJobParams{
		UserID: pgUserID,
		JobID:  pgJobID,
		Notes:  pgtype.Text{String: req.Notes, Valid: req.Notes != ""},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Job saved successfully"})
}

// DELETE /api/v1/jobs/{id}/save — Unsave/unbookmark a job
func (c *SavedJobController) UnsaveJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	pgUserID := uuidToPgtype(userID)
	pgJobID := uuidToPgtype(jobID)

	err = c.queries.UnsaveJob(r.Context(), database.UnsaveJobParams{
		UserID: pgUserID,
		JobID:  pgJobID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to unsave job: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Job unsaved successfully"})
}

// GET /api/v1/jobs/saved — List all saved jobs for the authenticated user
func (c *SavedJobController) ListSavedJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, offset := parsePagination(r)
	pgUserID := uuidToPgtype(userID)

	savedJobs, err := c.queries.GetSavedJobsByUser(r.Context(), database.GetSavedJobsByUserParams{
		UserID: pgUserID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch saved jobs: "+err.Error())
		return
	}

	count, _ := c.queries.CountSavedJobsByUser(r.Context(), pgUserID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"saved_jobs": savedJobs,
		"total":      count,
		"limit":      limit,
		"offset":     offset,
	})
}

// GET /api/v1/jobs/{id}/saved — Check if a job is saved by the user
func (c *SavedJobController) IsJobSaved(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	pgUserID := uuidToPgtype(userID)
	pgJobID := uuidToPgtype(jobID)

	isSaved, err := c.queries.IsJobSaved(r.Context(), database.IsJobSavedParams{
		UserID: pgUserID,
		JobID:  pgJobID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check saved status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"is_saved": isSaved})
}

// helper to convert uuid.UUID → pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	var pg pgtype.UUID
	copy(pg.Bytes[:], id[:])
	pg.Valid = true
	return pg
}
