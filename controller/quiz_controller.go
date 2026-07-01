package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Izone-hub/talent-backend/service"
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

	// Decode from Request Body (This is the dynamic way!)
	var req struct {
		ApplicationID string `json:"application_id"`
		JobID         string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Use the IDs from the request body
	err := c.quizService.StartQuizAttempt(r.Context(), id, claims.UserID.String(), req.ApplicationID, req.JobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to start quiz: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Quiz started successfully"})
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
		if err.Error() == "no rows in result set" {
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "finished",
				"message": "No more questions available or quiz completed",
			})
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to get question: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, question)
}

// 5. SaveAnswer handler
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
        writeError(w, http.StatusInternalServerError, "Failed to save answer: "+err.Error())
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

	result, err := c.quizService.RunQuizCode(r.Context(), id, req.QuestionID, req.Language, req.Code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
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
		writeError(w, http.StatusNotFound, "Quiz attempt context missing: "+err.Error())
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
		writeError(w, http.StatusInternalServerError, "Failed to submit quiz: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Quiz submitted successfully"})
}

// --- Uniform Helper Fallbacks ---
// If these are defined in another file within package controller, delete them from here.
