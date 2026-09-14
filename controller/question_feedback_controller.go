package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type QuestionFeedbackController struct {
	feedbackService *service.QuestionFeedbackService
}

func NewQuestionFeedbackController(feedbackService *service.QuestionFeedbackService) *QuestionFeedbackController {
	return &QuestionFeedbackController{feedbackService: feedbackService}
}

func (c *QuestionFeedbackController) userAndQuestion(r *http.Request) (*service.Claims, uuid.UUID, bool) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		return nil, uuid.Nil, false
	}
	questionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return claims, uuid.Nil, false
	}
	return claims, questionID, true
}

func (c *QuestionFeedbackController) Get(w http.ResponseWriter, r *http.Request) {
	claims, questionID, ok := c.userAndQuestion(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid question or user")
		return
	}
	feedback, err := c.feedbackService.Get(r.Context(), claims.UserID, questionID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"feedback": ""})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"feedback": feedback})
}

func (c *QuestionFeedbackController) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, questionID, ok := c.userAndQuestion(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid question or user")
		return
	}
	var req struct {
		Feedback string `json:"feedback"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	validFeedback := map[string]bool{"like": true, "dislike": true, "not_related": true, "not_enough_time": true, "too_difficult": true, "unclear_question": true}
	if !validFeedback[req.Feedback] {
		writeError(w, http.StatusBadRequest, "Invalid feedback value")
		return
	}
	feedback, err := c.feedbackService.Upsert(r.Context(), claims.UserID, questionID, req.Feedback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save question feedback")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"feedback": feedback})
}
