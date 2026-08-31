package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

// SurveyQuestionController handles HTTP requests for job screening questions.
type SurveyQuestionController struct {
	surveyService *service.SurveyQuestionService
	jobService    *service.JobService
}

// NewSurveyQuestionController creates a new SurveyQuestionController.
func NewSurveyQuestionController(
	surveyService *service.SurveyQuestionService,
	jobService *service.JobService,
) *SurveyQuestionController {
	return &SurveyQuestionController{
		surveyService: surveyService,
		jobService:    jobService,
	}
}

// PUT /api/v1/jobs/{id}/survey-questions — Replace all screening questions for a job
func (c *SurveyQuestionController) UpsertQuestions(w http.ResponseWriter, r *http.Request) {
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

	// Verify the job exists and belongs to the user
	job, err := c.jobService.GetJob(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if job.PostedBy != userID {
		writeError(w, http.StatusForbidden, "You can only edit your own jobs")
		return
	}

	var req models.UpsertSurveyQuestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	questions, err := c.surveyService.UpsertQuestions(r.Context(), jobID, req.Questions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save survey questions: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"questions": questions,
		"total":     len(questions),
	})
}

// GET /api/v1/jobs/{id}/survey-questions — Get all screening questions for a job
func (c *SurveyQuestionController) GetQuestions(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	questions, err := c.surveyService.GetQuestions(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch survey questions: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"questions": questions,
		"total":     len(questions),
	})
}

// DELETE /api/v1/jobs/{jobID}/survey-questions/{questionID} — Remove a single question
func (c *SurveyQuestionController) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
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

	questionIDStr := r.PathValue("questionID")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid question ID")
		return
	}

	// Verify ownership
	job, err := c.jobService.GetJob(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Job not found")
		return
	}
	if job.PostedBy != userID {
		writeError(w, http.StatusForbidden, "You can only edit your own jobs")
		return
	}

	if err := c.surveyService.DeleteQuestion(r.Context(), jobID, questionID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete question: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Question deleted"})
}

// POST /api/v1/jobs/{id}/apply-survey — Candidate submits survey answers before applying
func (c *SurveyQuestionController) SubmitSurveyAnswers(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*service.Claims)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Parse the candidate's answers
	var req struct {
		Answers map[string]bool `json:"answers"` // question_id -> answer (true=Yes, false=No)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Fetch the job's survey questions
	questions, err := c.surveyService.GetQuestions(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch survey questions")
		return
	}

	// If no questions, auto-pass
	if len(questions) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"passed":   true,
			"message":  "No screening questions — you may proceed to apply",
			"results":  []interface{}{},
		})
		return
	}

	// Validate each answer
	var results []map[string]interface{}
	allPassed := true
	for _, q := range questions {
		qID := q.ID.String()
		answer, exists := req.Answers[qID]
		passed := exists && answer == q.ExpectedAnswer
		if !passed {
			allPassed = false
		}
		results = append(results, map[string]interface{}{
			"question_id":   qID,
			"question_text": q.QuestionText,
			"expected":      q.ExpectedAnswer,
			"your_answer":   answer,
			"passed":        passed,
		})
	}

	_ = claims // available if you need user info for logging

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"passed":  allPassed,
		"message": surveyResultMessage(allPassed, len(questions)),
		"results": results,
	})
}

func surveyResultMessage(passed bool, total int) string {
	if passed {
		return "You passed all screening questions. You may proceed to apply."
	}
	return "You did not pass all screening questions. Please review your answers."
}
