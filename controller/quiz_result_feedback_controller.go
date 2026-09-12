package controller

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type QuizResultFeedbackController struct {
	feedbackService *service.QuizResultFeedbackService
}

func NewQuizResultFeedbackController(feedbackService *service.QuizResultFeedbackService) *QuizResultFeedbackController {
	return &QuizResultFeedbackController{feedbackService: feedbackService}
}

func (c *QuizResultFeedbackController) userAndQuizAttempt(r *http.Request) (*service.Claims, uuid.UUID, bool) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		return nil, uuid.Nil, false
	}
	quizAttemptID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return claims, uuid.Nil, false
	}
	return claims, quizAttemptID, true
}

func (c *QuizResultFeedbackController) Get(w http.ResponseWriter, r *http.Request) {
	claims, quizAttemptID, ok := c.userAndQuizAttempt(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt or user")
		return
	}

	var feedback *service.QuizResultFeedbackResponse
	var err error

	if claims.Role == "admin" {
		if uidStr := r.URL.Query().Get("user_id"); uidStr != "" {
			if targetUID, parseErr := uuid.Parse(uidStr); parseErr == nil {
				feedback, err = c.feedbackService.Get(r.Context(), targetUID, quizAttemptID)
			}
		}
		if feedback == nil {
			feedback, err = c.feedbackService.GetByAttempt(r.Context(), quizAttemptID)
		}
	} else {
		feedback, err = c.feedbackService.Get(r.Context(), claims.UserID, quizAttemptID)
	}

	if err != nil || feedback == nil {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, feedback)
}

func (c *QuizResultFeedbackController) GetByApplication(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}
	feedback, err := c.feedbackService.GetByApplication(r.Context(), appID)
	if err != nil || feedback == nil {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, feedback)
}

func (c *QuizResultFeedbackController) Upsert(w http.ResponseWriter, r *http.Request) {
	claims, quizAttemptID, ok := c.userAndQuizAttempt(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt or user")
		return
	}
	var req struct {
		Rating  string `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Rating == "" {
		req.Rating = "positive"
	}
	if req.Rating != "positive" && req.Rating != "negative" {
		writeError(w, http.StatusBadRequest, "Rating must be positive or negative")
		return
	}
	log.Printf("[QuizResultFeedback] Upserting feedback: user=%s attempt=%s rating=%s comment_len=%d", claims.UserID, quizAttemptID, req.Rating, len(req.Comment))
	feedback, err := c.feedbackService.Upsert(r.Context(), claims.UserID, quizAttemptID, req.Rating, req.Comment)
	if err != nil {
		if strings.Contains(err.Error(), "does not belong") {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "after quiz completion") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, service.ErrFeedbackAlreadySubmitted) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrNonEnglishText) || errors.Is(err, service.ErrProfanityDetected) || strings.Contains(err.Error(), "Feedback contains") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[QuizResultFeedback] Upsert error: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save feedback")
		return
	}
	writeJSON(w, http.StatusOK, feedback)
}

func (c *QuizResultFeedbackController) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := service.ValidateFeedbackComment(req.Comment); err != nil {
		unreadable := service.FindUnreadableWords(req.Comment)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":            false,
			"error":            err.Error(),
			"unreadable_words": unreadable,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":            true,
		"unreadable_words": []string{},
	})
}

func (c *QuizResultFeedbackController) Delete(w http.ResponseWriter, r *http.Request) {
	claims, quizAttemptID, ok := c.userAndQuizAttempt(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid quiz attempt or user")
		return
	}
	if err := c.feedbackService.Delete(r.Context(), claims.UserID, quizAttemptID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete feedback")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
