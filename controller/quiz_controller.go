package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type QuizController struct {
	quizService *service.QuizService
}

func NewQuizController(qs *service.QuizService) *QuizController {
	return &QuizController{quizService: qs}
}

// 1. ListQuizzes handler
func (c *QuizController) ListQuizzes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized user context extraction failed")
		return
	}

	quizzes, err := c.quizService.GetUserQuizzes(r.Context(), claims.UserID.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query quizzes: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, quizzes)
}

// 2. GetQuiz handler
func (c *QuizController) GetQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	quiz, err := c.quizService.GetQuizAttempt(r.Context(), id, claims.UserID.String())
	if err != nil {
		writeError(w, http.StatusNotFound, "Quiz not found: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, quiz)
}

// 3. StartQuiz handler
func (c *QuizController) StartQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	// Decode from Request Body (optional if quiz attempt already exists)
	var req struct {
		ApplicationID string `json:"application_id"`
		JobID         string `json:"job_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Use the IDs from the request body
	resp, err := c.quizService.StartQuizAttempt(r.Context(), id, claims.UserID.String(), req.ApplicationID, req.JobID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "does not belong"):
			writeError(w, http.StatusForbidden, err.Error())
		case strings.Contains(err.Error(), "not found"):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, "Failed to start quiz: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// 4. GetNextQuestion handler (Replaces the bulk list endpoint)
func (c *QuizController) GetNextQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	quizID := r.PathValue("id")
	log.Printf("GetNextQuestion called: quizID=%s, userID=%s", quizID, claims.UserID.String())

	question, err := c.quizService.GetNextQuestion(r.Context(), quizID, claims.UserID.String())
	if err != nil {
		log.Printf("GetNextQuestion error: %v", err)
		switch {
		case strings.Contains(err.Error(), "does not belong"):
			writeError(w, http.StatusForbidden, "You do not own this quiz attempt")
		case strings.Contains(err.Error(), "not found"):
			writeError(w, http.StatusNotFound, "Quiz attempt not found")
		case err.Error() == "no rows in result set":
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "finished",
				"message": "No more questions available or quiz completed",
			})
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get question: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// 5. SaveAnswer handler
func (c *QuizController) SaveAnswer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	var req struct {
		QuestionID       string `json:"question_id"`
		UserAnswer       string `json:"user_answer"`
		TimeSpentSeconds int    `json:"time_spent_seconds"`
		IsSkipped        bool   `json:"is_skipped"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Call the Service layer (which handles the DB check)
	err := c.quizService.SaveQuizAnswer(r.Context(), id, claims.UserID.String(), req.QuestionID, req.UserAnswer, req.TimeSpentSeconds, req.IsSkipped)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "does not belong to this user"):
			writeError(w, http.StatusForbidden, err.Error())
		case strings.Contains(err.Error(), "not found"):
			writeError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "out of order"),
			strings.Contains(err.Error(), "does not belong to this quiz attempt"),
			strings.Contains(err.Error(), "no longer active"),
			strings.Contains(err.Error(), "time limit exceeded"),
			strings.Contains(err.Error(), "completed"):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to save answer: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Answer saved successfully"})
}

// 6. RunCode handler (for coding_challenge questions)
func (c *QuizController) RunCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	var req struct {
		QuestionID string `json:"question_id"`
		Language   string `json:"language"`
		Code       string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if req.QuestionID == "" {
		writeError(w, http.StatusBadRequest, "question_id is required")
		return
	}

	log.Printf("RunCode: attemptID=%s, userID=%s, questionID=%s, lang=%s", id, claims.UserID.String(), req.QuestionID, req.Language)

	_, err := c.quizService.RunQuizCode(r.Context(), id, claims.UserID.String(), req.QuestionID, req.Language, req.Code)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "does not belong"):
			writeError(w, http.StatusForbidden, err.Error())
		case strings.Contains(err.Error(), "not found"):
			writeError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "not active"), strings.Contains(err.Error(), "completed"):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "executed",
		"message": "Code executed successfully",
	})
}

// 7. SubmitQuiz handler
func (c *QuizController) SubmitQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	quiz, err := c.quizService.GetQuizAttempt(r.Context(), id, claims.UserID.String())
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "does not belong"):
			writeError(w, http.StatusForbidden, "You do not own this quiz attempt")
		case strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows"):
			writeError(w, http.StatusNotFound, "Quiz attempt not found")
		default:
			writeError(w, http.StatusNotFound, "Quiz attempt context missing: "+err.Error())
		}
		return
	}

	var targetTags []string
	if quiz.Type != "" && quiz.Type != "General" {
		tagsRaw := strings.Split(quiz.Type, ",")
		for _, t := range tagsRaw {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				targetTags = append(targetTags, trimmed)
			}
		}
	}

	err = c.quizService.SubmitQuizAttempt(r.Context(), id, claims.UserID.String(), targetTags)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "does not belong"):
			writeError(w, http.StatusForbidden, err.Error())
		case strings.Contains(err.Error(), "already completed"), strings.Contains(err.Error(), "cannot be submitted"):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to submit quiz: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Quiz submitted successfully"})
}

// GetQuizResult handler returns the lightweight initial summary of a completed quiz attempt.
func (c *QuizController) GetQuizResult(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	isAdmin := claims.Role == "admin"

	summary, err := c.quizService.GetQuizResultSummary(r.Context(), id, claims.UserID.String(), isAdmin)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid attempt ID"),
			err.Error() == "no rows in result set":
			writeError(w, http.StatusNotFound, "Quiz attempt not found: "+err.Error())
		case strings.Contains(err.Error(), "does not belong"),
			strings.Contains(err.Error(), "only available after quiz completion"):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get quiz result: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetQuizReview handler returns the lightweight initial summary of a completed quiz attempt,
// or question review items if requested via view=questions or questions=true.
func (c *QuizController) GetQuizReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	isAdmin := claims.Role == "admin"

	if r.URL.Query().Get("view") == "questions" || r.URL.Query().Get("questions") == "true" {
		questions, err := c.quizService.GetQuizReviewQuestions(r.Context(), id, claims.UserID.String(), isAdmin)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "invalid attempt ID"),
				err.Error() == "no rows in result set":
				writeError(w, http.StatusNotFound, "Quiz attempt not found: "+err.Error())
			case strings.Contains(err.Error(), "does not belong"),
				strings.Contains(err.Error(), "only available after quiz completion"):
				writeError(w, http.StatusForbidden, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "Failed to get quiz review questions: "+err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, questions)
		return
	}

	summary, err := c.quizService.GetQuizResultSummary(r.Context(), id, claims.UserID.String(), isAdmin)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid attempt ID"),
			err.Error() == "no rows in result set":
			writeError(w, http.StatusNotFound, "Quiz attempt not found: "+err.Error())
		case strings.Contains(err.Error(), "does not belong"),
			strings.Contains(err.Error(), "only available after quiz completion"):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get quiz review: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// GetQuizReviewQuestions handler returns the lightweight question review list.
func (c *QuizController) GetQuizReviewQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	isAdmin := claims.Role == "admin"

	questions, err := c.quizService.GetQuizReviewQuestions(r.Context(), id, claims.UserID.String(), isAdmin)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid attempt ID"),
			err.Error() == "no rows in result set":
			writeError(w, http.StatusNotFound, "Quiz attempt not found: "+err.Error())
		case strings.Contains(err.Error(), "does not belong"),
			strings.Contains(err.Error(), "only available after quiz completion"):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get quiz review questions: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, questions)
}

// GetQuizQuestionDetail handler returns details (options, correct answer, explanation, code output) for a single question.
func (c *QuizController) GetQuizQuestionDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	questionID := r.PathValue("questionId")
	isAdmin := claims.Role == "admin"

	detail, err := c.quizService.GetQuizQuestionDetail(r.Context(), id, questionID, claims.UserID.String(), isAdmin)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid attempt ID"),
			strings.Contains(err.Error(), "invalid question ID"),
			err.Error() == "no rows in result set":
			writeError(w, http.StatusNotFound, "Question not found: "+err.Error())
		case strings.Contains(err.Error(), "does not belong"),
			strings.Contains(err.Error(), "only available after quiz completion"):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Failed to get question detail: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// ListJobQuizzes handler lists all quiz attempts taken for a job
// (admin only), including applicant data and quiz results.
func (c *QuizController) ListJobQuizzes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	quizzes, err := c.quizService.GetJobQuizAttempts(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list quizzes: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, quizzes)
}

// ListUserQuizzes handler lists all quiz attempts taken by a user across all
// jobs (admin only), including job info, applicant data and quiz results.
func (c *QuizController) ListUserQuizzes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	quizzes, err := c.quizService.GetUserQuizAttempts(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list quizzes: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, quizzes)
}

// --- Uniform Helper Fallbacks ---
// If these are defined in another file within package controller, delete them from here.
