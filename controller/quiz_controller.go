package controller

import (
	"encoding/json"
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

func (c *QuizController) StartQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")
	err := c.quizService.StartQuizAttempt(r.Context(), id, claims.UserID.String())
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to start quiz: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Quiz started successfully"})
}

func (c *QuizController) GetQuizQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	// ታጎቹን ለማግኘት በመጀመሪያ የ Quiz Attempt መረጃን እናነባለን (ይህም የጀሚናይ ማይክሮሰርቪስ የመረጣቸውን ታጎች የያዘ ነው)
	quiz, err := c.quizService.GetQuizAttempt(r.Context(), id, claims.UserID.String())
	if err != nil {
		writeError(w, http.StatusNotFound, "Quiz attempt context missing: "+err.Error())
		return
	}

	// በኮማ የተለዩትን ታጎች ወደ []string እንቀይራቸዋለን
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

	// የተጣሩትን ታጎች ለሰርቪሱ እንሰጣለን
	questions, err := c.quizService.GetQuizQuestions(r.Context(), id, claims.UserID.String(), targetTags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch quiz questions: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, questions)
}

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

	err := c.quizService.SaveQuizAnswer(r.Context(), id, claims.UserID.String(), req.QuestionID, req.UserAnswer, req.TimeSpentSeconds, req.IsSkipped)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save answer: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Answer saved successfully"})
}

func (c *QuizController) SubmitQuiz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id := r.PathValue("id")

	// በስሌቱ ወቅት ጥያቄዎቹን በድጋሚ ለማጣራት የ Attempt ታጎችን እናወጣለን
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

	// የዘመነውን ሰርቪስ በአዲሱ አርጉመንት እንጠራዋለን
	err = c.quizService.SubmitQuizAttempt(r.Context(), id, claims.UserID.String(), targetTags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit quiz: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Quiz submitted successfully"})
}

// ዩኒፎርም እንዲሆን የተጠቀሱት writeError እና writeJSON ረዳት ፋንክሽኖች (በሌላ ቦታ ካልተገለጹ)

