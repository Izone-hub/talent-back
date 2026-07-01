package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type TagController struct {
	tagService *service.TagService
}

func NewTagController(tagService *service.TagService) *TagController {
	return &TagController{
		tagService: tagService,
	}
}

// POST /tags
func (c *TagController) CreateTag(w http.ResponseWriter, r *http.Request) {
	// Usually only admins can create tags, assumed middleware checks it.

	var req models.Tag
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Tag name is required")
		return
	}

	tag, err := c.tagService.CreateTag(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, tag)
}

// GET /tags
func (c *TagController) ListTags(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	tags, err := c.tagService.ListTags(r.Context(), int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tags":   tags,
		"limit":  limit,
		"offset": offset,
	})
}

// GET /tags/{id}
func (c *TagController) GetTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tagID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid tag ID format")
		return
	}

	tag, err := c.tagService.GetTagByID(r.Context(), tagID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tag not found")
		return
	}

	writeJSON(w, http.StatusOK, tag)
}

func (c *TagController) AssignTagToJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID   uuid.UUID `json:"job_id"`
		TagID   uuid.UUID `json:"tag_id"`
		TagName string    `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.JobID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	tag, err := c.tagService.AssignTagToJob(r.Context(), req.JobID, req.TagID, req.TagName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Tag assigned to job successfully",
		"tag":     tag,
	})
}

func (c *TagController) GetJobTags(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID format")
		return
	}

	tags, err := c.tagService.GetJobTags(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tags)
}

func (c *TagController) RemoveTagFromJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID   uuid.UUID `json:"job_id"`
		TagID   uuid.UUID `json:"tag_id"`
		TagName string    `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.JobID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	err := c.tagService.RemoveTagFromJob(r.Context(), req.JobID, req.TagID, req.TagName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Tag removed from job successfully",
	})
}

func (c *TagController) GetTagJobs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tagID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid tag ID format")
		return
	}

	limit, offset := parsePagination(r)

	jobs, err := c.tagService.GetTagJobs(r.Context(), tagID, int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]models.JobResponse, 0, len(jobs))
	for _, j := range jobs {
		responses = append(responses, j.ToResponse())
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":   responses,
		"limit":  limit,
		"offset": offset,
	})
}

// PUT /tags/{id}
func (c *TagController) UpdateTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tagID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid tag ID format")
		return
	}

	var req models.Tag
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	tag, err := c.tagService.UpdateTag(r.Context(), tagID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tag)
}

// DELETE /tags/{id}
func (c *TagController) DeleteTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tagID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid tag ID format")
		return
	}

	err = c.tagService.DeleteTag(r.Context(), tagID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Tag deleted successfully",
	})
}
