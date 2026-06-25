package controller

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
)

// QuestionController handles HTTP requests for questions.
type QuestionController struct {
	questionService *service.QuestionService
}

// NewQuestionController creates a new QuestionController.
func NewQuestionController(questionService *service.QuestionService) *QuestionController {
	return &QuestionController{
		questionService: questionService,
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/questions — Create a new question
// ---------------------------------------------------------------------------
func (c *QuestionController) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	question, err := c.questionService.CreateQuestion(r.Context(), userID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, question)
}

// ---------------------------------------------------------------------------
// GET /api/v1/questions — List questions
// ---------------------------------------------------------------------------
func (c *QuestionController) ListQuestions(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	questions, err := c.questionService.ListQuestions(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"questions": questions,
		"total":     len(questions),
		"limit":     limit,
		"offset":    offset,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v1/questions/{id} — Get a single question
// ---------------------------------------------------------------------------
func (c *QuestionController) GetQuestion(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	question, err := c.questionService.GetQuestion(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Question not found")
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// ---------------------------------------------------------------------------
// PUT /api/v1/questions/{id} — Update a question
// ---------------------------------------------------------------------------
func (c *QuestionController) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	var req models.UpdateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	question, err := c.questionService.UpdateQuestion(r.Context(), userID, id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/questions/{id} — Delete a question
// ---------------------------------------------------------------------------
func (c *QuestionController) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	if err := c.questionService.DeleteQuestion(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
