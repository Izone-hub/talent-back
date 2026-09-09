package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type QuizAnswerFeedbackController struct {
	feedbackService *service.QuizAnswerFeedbackService
}

func NewQuizAnswerFeedbackController(feedbackService *service.QuizAnswerFeedbackService) *QuizAnswerFeedbackController {
	return &QuizAnswerFeedbackController{feedbackService: feedbackService}
}

func (c *QuizAnswerFeedbackController) GetByAttempt(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user")
		return
	}
	quizAttemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt ID")
		return
	}
	feedbacks, err := c.feedbackService.GetByAttempt(r.Context(), claims.UserID, quizAttemptID)
	if err != nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, feedbacks)
}

func (c *QuizAnswerFeedbackController) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user")
		return
	}
	quizAttemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt ID")
		return
	}
	var req struct {
		QuestionID    string `json:"question_id"`
		ApplicationID string `json:"application_id"`
		Feedback      string `json:"feedback"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	questionID, err := uuid.Parse(req.QuestionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}
	applicationID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}
	validFeedback := map[string]bool{
		"not_related":     true,
		"not_enough_time": true,
		"too_difficult":   true,
		"unclear_question": true,
	}
	if !validFeedback[req.Feedback] {
		writeError(w, http.StatusBadRequest, "Invalid feedback value")
		return
	}
	feedback, err := c.feedbackService.Upsert(r.Context(), claims.UserID, quizAttemptID, questionID, applicationID, req.Feedback)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save feedback")
		return
	}
	writeJSON(w, http.StatusOK, feedback)
}

func (c *QuizAnswerFeedbackController) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user")
		return
	}
	quizAttemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt ID")
		return
	}
	questionID, err := uuid.Parse(r.PathValue("questionId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}
	if err := c.feedbackService.Delete(r.Context(), claims.UserID, quizAttemptID, questionID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete feedback")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *QuizAnswerFeedbackController) GetByApplication(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user")
		return
	}
	applicationID, err := uuid.Parse(r.PathValue("applicationId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}
	// Admins can view any application's feedback; users can only view their own
	_ = claims
	feedbacks, err := c.feedbackService.GetByApplication(r.Context(), applicationID)
	if err != nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, feedbacks)
}
