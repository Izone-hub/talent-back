package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuizAttempt represents a quiz attempt structure in the database
type QuizAttempt struct {
	ID               uuid.UUID `json:"id"`
	ApplicationID    uuid.UUID `json:"application_id"`
	JobID            uuid.UUID `json:"job_id"`
	UserID           uuid.UUID `json:"user_id"`
	Title            string    `json:"title"`
	Type             string    `json:"type"` // Tags for Gemini microservice (comma-separated)
	Status           string    `json:"status"`
	QuestionsPerQuiz int32     `json:"questions_per_quiz"`
	TotalQuestions   int32     `json:"total_questions"`
	CreatedAt        time.Time `json:"created_at"`
}

// QuizService handles all quiz logic and database calls
type QuizService struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

// NewQuizService receives the *pgxpool.Pool sent from main.go
func NewQuizService(pool *pgxpool.Pool) *QuizService {
	return &QuizService{pool: pool, queries: database.New(pool)}
}

// JobQuizApplicant holds the applicant data attached to an admin quiz view.
type JobQuizApplicant struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	GithubUsername string `json:"github_username"`
	AvatarURL      string `json:"avatar_url"`
}

// JobQuizResults holds the optional AI-generated results of a quiz attempt.
type JobQuizResults struct {
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
	AIFeedback string   `json:"ai_feedback"`
}

// JobQuizAttempt is the admin view of a quiz attempt taken for a job.
type JobQuizAttempt struct {
	ID               uuid.UUID        `json:"id"`
	ApplicationID    uuid.UUID        `json:"application_id"`
	JobApplicationID uuid.UUID        `json:"job_application_id"`
	UserID           uuid.UUID        `json:"user_id"`
	ClientID         uuid.UUID        `json:"client_id"`
	JobID            uuid.UUID        `json:"job_id"`
	JobTitle         string           `json:"job_title,omitempty"`
	JobCompany       string           `json:"job_company,omitempty"`
	Status           string           `json:"status"`
	Score            *int32           `json:"score"`
	Passed           *bool            `json:"passed"`
	CorrectAnswers   *int32           `json:"correct_answers"`
	QuestionsPerQuiz int32            `json:"questions_per_quiz"`
	StartedAt        *time.Time       `json:"started_at"`
	CompletedAt      *time.Time       `json:"completed_at"`
	TimeSpentSeconds *int32           `json:"time_spent_seconds"`
	Applicant        JobQuizApplicant `json:"applicant"`
	QuizResults      *JobQuizResults  `json:"quiz_results,omitempty"`
}

// GetJobQuizAttempts lists all quiz attempts taken for a given job (admin
// view), including the applicant data and optional AI quiz results.
func (s *QuizService) GetJobQuizAttempts(ctx context.Context, jobID string, limit, offset int32) ([]JobQuizAttempt, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobUUID[:])
	pgJobID.Valid = true

	rows, err := s.queries.ListQuizAttemptsByJob(ctx, database.ListQuizAttemptsByJobParams{
		JobID:  pgJobID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	attempts := make([]JobQuizAttempt, 0, len(rows))
	for _, row := range rows {
		attempt := JobQuizAttempt{
			Status:           string(row.Status),
			Score:            int4Ptr(row.Score),
			Passed:           boolPtr(row.Passed),
			CorrectAnswers:   int4Ptr(row.CorrectAnswers),
			QuestionsPerQuiz: row.QuestionsPerQuiz,
			StartedAt:        timestampPtr(row.StartedAt),
			CompletedAt:      timestampPtr(row.CompletedAt),
			TimeSpentSeconds: int4Ptr(row.TimeSpentSeconds),
			Applicant: JobQuizApplicant{
				Name:           row.Name.String,
				Email:          row.Email.String,
				GithubUsername: row.GithubUsername,
				AvatarURL:      row.AvatarUrl.String,
			},
		}

		if id, err := uuid.FromBytes(row.QuizID.Bytes[:]); err == nil {
			attempt.ID = id
		}
		if id, err := uuid.FromBytes(row.ApplicationID.Bytes[:]); err == nil {
			attempt.ApplicationID = id
			attempt.JobApplicationID = id
		}
		if id, err := uuid.FromBytes(row.ClientID.Bytes[:]); err == nil {
			attempt.ClientID = id
			attempt.UserID = id
		}
		if id, err := uuid.FromBytes(row.JobID.Bytes[:]); err == nil {
			attempt.JobID = id
		}

		if row.Strengths != nil || row.Weaknesses != nil || row.AiFeedback.Valid {
			attempt.QuizResults = &JobQuizResults{
				Strengths:  row.Strengths,
				Weaknesses: row.Weaknesses,
				AIFeedback: row.AiFeedback.String,
			}
		}

		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

// GetUserQuizAttempts lists all quiz attempts taken by a given user across
// all jobs (admin view), including job info, applicant data and optional AI
// quiz results.
func (s *QuizService) GetUserQuizAttempts(ctx context.Context, userID string, limit, offset int32) ([]JobQuizAttempt, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], userUUID[:])
	pgUserID.Valid = true

	rows, err := s.queries.ListQuizAttemptsByUser(ctx, database.ListQuizAttemptsByUserParams{
		UserID: pgUserID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	attempts := make([]JobQuizAttempt, 0, len(rows))
	for _, row := range rows {
		attempt := JobQuizAttempt{
			JobTitle:         row.JobTitle,
			JobCompany:       row.JobCompany,
			Status:           string(row.Status),
			Score:            int4Ptr(row.Score),
			Passed:           boolPtr(row.Passed),
			CorrectAnswers:   int4Ptr(row.CorrectAnswers),
			QuestionsPerQuiz: row.QuestionsPerQuiz,
			StartedAt:        timestampPtr(row.StartedAt),
			CompletedAt:      timestampPtr(row.CompletedAt),
			TimeSpentSeconds: int4Ptr(row.TimeSpentSeconds),
			Applicant: JobQuizApplicant{
				Name:           row.Name.String,
				Email:          row.Email.String,
				GithubUsername: row.GithubUsername,
				AvatarURL:      row.AvatarUrl.String,
			},
		}

		if id, err := uuid.FromBytes(row.QuizID.Bytes[:]); err == nil {
			attempt.ID = id
		}
		if id, err := uuid.FromBytes(row.ApplicationID.Bytes[:]); err == nil {
			attempt.ApplicationID = id
			attempt.JobApplicationID = id
		}
		if id, err := uuid.FromBytes(row.ClientID.Bytes[:]); err == nil {
			attempt.ClientID = id
			attempt.UserID = id
		}
		if id, err := uuid.FromBytes(row.JobID.Bytes[:]); err == nil {
			attempt.JobID = id
		}

		if row.Strengths != nil || row.Weaknesses != nil || row.AiFeedback.Valid {
			attempt.QuizResults = &JobQuizResults{
				Strengths:  row.Strengths,
				Weaknesses: row.Weaknesses,
				AIFeedback: row.AiFeedback.String,
			}
		}

		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

// QuizResultSummary is the lightweight initial response for a completed quiz attempt.
type QuizResultSummary struct {
	ID                uuid.UUID  `json:"id"`
	ApplicationID     uuid.UUID  `json:"application_id"`
	JobID             uuid.UUID  `json:"job_id"`
	JobTitle          string     `json:"job_title"`
	Company           string     `json:"company"`
	Status            string     `json:"status"`
	Score             *int32     `json:"score"`
	Passed            *bool      `json:"passed"`
	CorrectAnswers    *int32     `json:"correct_answers"`
	TotalQuestions    int32      `json:"total_questions"`
	AnsweredQuestions int32      `json:"answered_questions"`
	PassingScore      int32      `json:"passing_score"`
	StartedAt         *time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	TimeSpentSeconds  *int32     `json:"time_spent_seconds"`
}

// QuizReviewQuestionItem is a lightweight item in the question review list without answer keys.
type QuizReviewQuestionItem struct {
	QuestionID     uuid.UUID `json:"question_id"`
	QuestionNumber int32     `json:"question_number"`
	QuestionText   string    `json:"question_text"`
	QuestionType   string    `json:"question_type"`
	Difficulty     string    `json:"difficulty"`
	UserAnswer     *string   `json:"user_answer"`
	IsCorrect      *bool     `json:"is_correct"`
	IsSkipped      bool      `json:"is_skipped"`
	Points         *int32    `json:"points,omitempty"`
}

// QuizQuestionDetail is fetched on-demand when a user opens a specific question for review.
type QuizQuestionDetail struct {
	QuestionID       uuid.UUID       `json:"question_id"`
	QuestionNumber   int32           `json:"question_number"`
	QuestionText     string          `json:"question_text"`
	QuestionType     string          `json:"question_type"`
	Difficulty       string          `json:"difficulty"`
	Points           *int32          `json:"points,omitempty"`
	Options          json.RawMessage `json:"options"`
	UserAnswer       *string         `json:"user_answer"`
	CorrectAnswer    *string         `json:"correct_answer,omitempty"`
	Explanation      *string         `json:"explanation,omitempty"`
	IsCorrect        *bool           `json:"is_correct"`
	IsSkipped        bool            `json:"is_skipped"`
	TimeSpentSeconds int32           `json:"time_spent_seconds"`
	CodeOutput       *string         `json:"code_output,omitempty"`
}

// GetQuizResultSummary returns the lightweight initial result summary for a completed quiz attempt.
func (s *QuizService) GetQuizResultSummary(ctx context.Context, attemptID, requesterID string, isAdmin bool) (*QuizResultSummary, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}

	var (
		appID, userID, jobID uuid.UUID
		jobTitle, company    string
		status               string
		score                pgtype.Int4
		passed               pgtype.Bool
		correctAnswers       pgtype.Int4
		questionsPerQuiz     int32
		totalQuestions       int32
		passingScore         int32
		startedAt            pgtype.Timestamp
		completedAt          pgtype.Timestamp
		timeSpent            pgtype.Int4
		answeredQuestions    int32
	)

	err = s.pool.QueryRow(ctx, `
		SELECT qa.application_id, qa.user_id, qa.job_id,
		       COALESCE(j.title, 'Quiz') AS job_title,
		       COALESCE(j.company, '') AS company,
		       qa.status, qa.score, qa.passed, qa.correct_answers,
		       qa.questions_per_quiz, qa.total_questions, qa.passing_score,
		       qa.started_at, qa.completed_at, qa.time_spent_seconds,
		       (SELECT COUNT(*)::int FROM quiz_answers WHERE quiz_attempt_id = qa.id) AS answered_questions
		FROM quiz_attempts qa
		LEFT JOIN jobs j ON qa.job_id = j.id
		WHERE qa.id = $1
	`, attemptUUID).Scan(
		&appID, &userID, &jobID, &jobTitle, &company, &status,
		&score, &passed, &correctAnswers,
		&questionsPerQuiz, &totalQuestions, &passingScore,
		&startedAt, &completedAt, &timeSpent, &answeredQuestions,
	)
	if err != nil {
		return nil, err
	}

	// Authorization: the attempt owner or an admin
	if !isAdmin {
		requesterUUID, err := uuid.Parse(requesterID)
		if err != nil {
			return nil, fmt.Errorf("invalid requester ID: %w", err)
		}
		if userID != requesterUUID {
			return nil, fmt.Errorf("quiz attempt does not belong to this user")
		}
		if status != "completed" {
			return nil, fmt.Errorf("quiz results are only available after quiz completion")
		}
	}

	totalQ := questionsPerQuiz
	if totalQ <= 0 {
		totalQ = totalQuestions
	}

	return &QuizResultSummary{
		ID:                attemptUUID,
		ApplicationID:     appID,
		JobID:             jobID,
		JobTitle:          jobTitle,
		Company:           company,
		Status:            status,
		Score:             int4Ptr(score),
		Passed:            boolPtr(passed),
		CorrectAnswers:    int4Ptr(correctAnswers),
		TotalQuestions:    totalQ,
		AnsweredQuestions: answeredQuestions,
		PassingScore:      passingScore,
		StartedAt:         timestampPtr(startedAt),
		CompletedAt:       timestampPtr(completedAt),
		TimeSpentSeconds:  int4Ptr(timeSpent),
	}, nil
}

// GetQuizReview returns the lightweight quiz result summary.
func (s *QuizService) GetQuizReview(ctx context.Context, attemptID, requesterID string, isAdmin bool) (*QuizResultSummary, error) {
	return s.GetQuizResultSummary(ctx, attemptID, requesterID, isAdmin)
}

// GetQuizReviewQuestions returns the lightweight question review list (no correct answers or explanations).
func (s *QuizService) GetQuizReviewQuestions(ctx context.Context, attemptID, requesterID string, isAdmin bool) ([]QuizReviewQuestionItem, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}

	var (
		userID uuid.UUID
		status string
	)
	err = s.pool.QueryRow(ctx, `
		SELECT user_id, status FROM quiz_attempts WHERE id = $1
	`, attemptUUID).Scan(&userID, &status)
	if err != nil {
		return nil, err
	}

	if !isAdmin {
		requesterUUID, err := uuid.Parse(requesterID)
		if err != nil {
			return nil, fmt.Errorf("invalid requester ID: %w", err)
		}
		if userID != requesterUUID {
			return nil, fmt.Errorf("quiz attempt does not belong to this user")
		}
		if status != "completed" {
			return nil, fmt.Errorf("quiz review is only available after quiz completion")
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT q.id,
		       COALESCE(qaq.question_order, 0) AS question_order,
		       q.question_text,
		       q.question_type,
		       q.difficulty,
		       q.points,
		       qa.user_answer,
		       qa.is_correct,
		       qa.is_skipped
		FROM quiz_answers qa
		JOIN questions q ON q.id = qa.question_id
		LEFT JOIN quiz_attempt_questions qaq ON qaq.quiz_attempt_id = qa.quiz_attempt_id AND qaq.question_id = qa.question_id
		WHERE qa.quiz_attempt_id = $1
		ORDER BY COALESCE(qaq.question_order, 9999), qa.created_at ASC
	`, attemptUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]QuizReviewQuestionItem, 0)
	idx := int32(1)
	for rows.Next() {
		var item QuizReviewQuestionItem
		var qOrder int32
		var points pgtype.Int4

		if err := rows.Scan(
			&item.QuestionID,
			&qOrder,
			&item.QuestionText,
			&item.QuestionType,
			&item.Difficulty,
			&points,
			&item.UserAnswer,
			&item.IsCorrect,
			&item.IsSkipped,
		); err != nil {
			return nil, err
		}

		if qOrder > 0 {
			item.QuestionNumber = qOrder
		} else {
			item.QuestionNumber = idx
		}
		item.Points = int4Ptr(points)

		items = append(items, item)
		idx++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// GetQuizQuestionDetail returns full details (options, correct answer, explanation, code output) for a single question.
func (s *QuizService) GetQuizQuestionDetail(ctx context.Context, attemptID, questionID, requesterID string, isAdmin bool) (*QuizQuestionDetail, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}
	qUUID, err := uuid.Parse(questionID)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %w", err)
	}

	var (
		userID uuid.UUID
		status string
	)
	err = s.pool.QueryRow(ctx, `
		SELECT user_id, status FROM quiz_attempts WHERE id = $1
	`, attemptUUID).Scan(&userID, &status)
	if err != nil {
		return nil, err
	}

	if !isAdmin {
		requesterUUID, err := uuid.Parse(requesterID)
		if err != nil {
			return nil, fmt.Errorf("invalid requester ID: %w", err)
		}
		if userID != requesterUUID {
			return nil, fmt.Errorf("quiz attempt does not belong to this user")
		}
		if status != "completed" {
			return nil, fmt.Errorf("question details are only available after quiz completion")
		}
	}

	var (
		detail           QuizQuestionDetail
		qOrder           int32
		points           pgtype.Int4
		optionsRaw       []byte
		timeSpentSeconds int32
		codeOutput       *string
	)

	err = s.pool.QueryRow(ctx, `
		SELECT q.id,
		       COALESCE(qaq.question_order, 1) AS question_order,
		       q.question_text,
		       q.question_type,
		       q.difficulty,
		       q.points,
		       q.options,
		       q.correct_answer,
		       q.explanation,
		       qa.user_answer,
		       qa.is_correct,
		       qa.is_skipped,
		       qa.time_spent_seconds,
		       qa.code_output
		FROM quiz_answers qa
		JOIN questions q ON q.id = qa.question_id
		LEFT JOIN quiz_attempt_questions qaq ON qaq.quiz_attempt_id = qa.quiz_attempt_id AND qaq.question_id = qa.question_id
		WHERE qa.quiz_attempt_id = $1 AND qa.question_id = $2
	`, attemptUUID, qUUID).Scan(
		&detail.QuestionID,
		&qOrder,
		&detail.QuestionText,
		&detail.QuestionType,
		&detail.Difficulty,
		&points,
		&optionsRaw,
		&detail.CorrectAnswer,
		&detail.Explanation,
		&detail.UserAnswer,
		&detail.IsCorrect,
		&detail.IsSkipped,
		&timeSpentSeconds,
		&codeOutput,
	)
	if err != nil {
		return nil, err
	}

	detail.QuestionNumber = qOrder
	detail.Points = int4Ptr(points)
	detail.TimeSpentSeconds = timeSpentSeconds

	if optionsRaw != nil {
		detail.Options = sanitizeQuizOptions(json.RawMessage(optionsRaw))
	} else {
		detail.Options = json.RawMessage(`[]`)
	}

	// For coding questions, return code_output if present and non-empty
	if detail.QuestionType == "coding_challenge" || (codeOutput != nil && *codeOutput != "") {
		detail.CodeOutput = codeOutput
	}

	return &detail, nil
}

// GetUserQuizzes retrieves all quiz attempts belonging to a specific user
func (s *QuizService) GetUserQuizzes(ctx context.Context, userID string) ([]QuizAttempt, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT qa.id, qa.user_id, COALESCE(j.title, 'Quiz') as title, qa.status, qa.created_at
		FROM quiz_attempts qa
		LEFT JOIN jobs j ON qa.job_id = j.id
		WHERE qa.user_id = $1
		ORDER BY qa.created_at DESC
	`, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quizzes []QuizAttempt
	for rows.Next() {
		var q QuizAttempt
		if err := rows.Scan(&q.ID, &q.UserID, &q.Title, &q.Status, &q.CreatedAt); err != nil {
			return nil, err
		}
		quizzes = append(quizzes, q)
	}

	if quizzes == nil {
		quizzes = []QuizAttempt{}
	}

	return quizzes, nil
}

// GetQuizAttempt fetches the attempt configuration and metadata, verifying user ownership or admin access
func (s *QuizService) GetQuizAttempt(ctx context.Context, attemptID string, userID string, isAdmin bool) (*QuizAttempt, error) {
	attUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var (
		qaUserID         uuid.UUID
		appID            uuid.UUID
		jobID            uuid.UUID
		jobTitle         string
		status           string
		questionsPerQuiz int32
		totalQuestions   int32
		createdAt        time.Time
	)

	err = s.pool.QueryRow(ctx, `
		SELECT qa.user_id, qa.application_id, qa.job_id, COALESCE(j.title, 'Technical Assessment'), qa.status, qa.questions_per_quiz, qa.total_questions, qa.created_at
		FROM quiz_attempts qa
		LEFT JOIN jobs j ON qa.job_id = j.id
		WHERE qa.id = $1
	`, attUUID).Scan(&qaUserID, &appID, &jobID, &jobTitle, &status, &questionsPerQuiz, &totalQuestions, &createdAt)
	if err != nil {
		return nil, err
	}

	if !isAdmin && qaUserID != userUUID {
		return nil, fmt.Errorf("quiz attempt does not belong to this user")
	}

	return &QuizAttempt{
		ID:               attUUID,
		ApplicationID:    appID,
		JobID:            jobID,
		UserID:           qaUserID,
		Title:            jobTitle,
		Type:             "General",
		Status:           status,
		QuestionsPerQuiz: questionsPerQuiz,
		TotalQuestions:   totalQuestions,
		CreatedAt:        createdAt,
	}, nil
}

// selectQuizQuestions selects exactly 10 questions for a job following the 4-tier distribution rules:
// Target distribution: 3 Easy, 3 Medium, 3 Hard, 1 Expert.
//
// Tier 1: Tag + requested difficulty
// Tier 2: Tag + any difficulty (for shortfall)
// Tier 3: General/untagged + requested difficulty (for shortfall)
// Tier 4: Any active question (for shortfall)
func (s *QuizService) selectQuizQuestions(ctx context.Context, jobID uuid.UUID) ([]uuid.UUID, error) {
	difficultyTargets := []struct {
		difficulty string
		count      int
	}{
		{"easy", 3},
		{"medium", 3},
		{"hard", 3},
		{"expert", 1},
	}
	const totalTarget = 10

	// 1. Fetch job tags (lowercase)
	tagRows, err := s.pool.Query(ctx, `
		SELECT LOWER(t.name) FROM tags t
		JOIN job_tags jt ON jt.tag_id = t.id
		WHERE jt.job_id = $1
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job tags: %w", err)
	}
	defer tagRows.Close()

	var jobTags []string
	for tagRows.Next() {
		var name string
		if err := tagRows.Scan(&name); err != nil {
			return nil, err
		}
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			jobTags = append(jobTags, trimmed)
		}
	}

	selectedIDs := make([]uuid.UUID, 0, totalTarget)
	selectedSet := make(map[uuid.UUID]bool)
	selectedByDiff := make(map[string]int)

	addSelected := func(ids []uuid.UUID) {
		for _, id := range ids {
			if !selectedSet[id] && len(selectedIDs) < totalTarget {
				selectedSet[id] = true
				selectedIDs = append(selectedIDs, id)
			}
		}
	}

	hasTags := len(jobTags) > 0

	if hasTags {
		// TIER 1: Tag + requested difficulty
		for _, dt := range difficultyTargets {
			needed := dt.count
			if needed <= 0 || len(selectedIDs) >= totalTarget {
				continue
			}

			rows, err := s.pool.Query(ctx, `
				SELECT q.id FROM questions q
				WHERE q.is_active = true
				  AND (q.question_type != 'coding_challenge' OR EXISTS (
				      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
				  ))
				  AND q.difficulty = $1
				  AND EXISTS (
				      SELECT 1 FROM unnest(q.tags) qt WHERE LOWER(qt) = ANY($2::text[])
				  )
				  AND ($3::uuid[] IS NULL OR cardinality($3::uuid[]) = 0 OR NOT (q.id = ANY($3::uuid[])))
				ORDER BY RANDOM()
				LIMIT $4
			`, dt.difficulty, jobTags, selectedIDs, needed)
			if err != nil {
				return nil, fmt.Errorf("tier 1 question selection failed for %s: %w", dt.difficulty, err)
			}

			var tier1IDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return nil, err
				}
				tier1IDs = append(tier1IDs, id)
			}
			rows.Close()
			selectedByDiff[dt.difficulty] += len(tier1IDs)
			addSelected(tier1IDs)
		}

		// TIER 2: Tag + any difficulty (fallback if still short of 10)
		if len(selectedIDs) < totalTarget {
			needed := totalTarget - len(selectedIDs)
			rows, err := s.pool.Query(ctx, `
				SELECT q.id, q.difficulty::text FROM questions q
				WHERE q.is_active = true
				  AND (q.question_type != 'coding_challenge' OR EXISTS (
				      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
				  ))
				  AND EXISTS (
				      SELECT 1 FROM unnest(q.tags) qt WHERE LOWER(qt) = ANY($1::text[])
				  )
				  AND ($2::uuid[] IS NULL OR cardinality($2::uuid[]) = 0 OR NOT (q.id = ANY($2::uuid[])))
				ORDER BY RANDOM()
				LIMIT $3
			`, jobTags, selectedIDs, needed)
			if err != nil {
				return nil, fmt.Errorf("tier 2 question selection failed: %w", err)
			}

			var tier2IDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				var diff string
				if err := rows.Scan(&id, &diff); err != nil {
					rows.Close()
					return nil, err
				}
				tier2IDs = append(tier2IDs, id)
				selectedByDiff[diff]++
			}
			rows.Close()
			addSelected(tier2IDs)
		}

		// TIER 3: General/untagged + requested difficulty (fallback if still short)
		if len(selectedIDs) < totalTarget {
			for _, dt := range difficultyTargets {
				neededDiff := dt.count - selectedByDiff[dt.difficulty]
				remainingSlots := totalTarget - len(selectedIDs)
				if neededDiff <= 0 || remainingSlots <= 0 {
					continue
				}
				if neededDiff > remainingSlots {
					neededDiff = remainingSlots
				}

				rows, err := s.pool.Query(ctx, `
					SELECT q.id FROM questions q
					WHERE q.is_active = true
					  AND (q.question_type != 'coding_challenge' OR EXISTS (
					      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
					  ))
					  AND q.difficulty = $1
					  AND NOT EXISTS (
					      SELECT 1 FROM unnest(q.tags) qt WHERE LOWER(qt) = ANY($2::text[])
					  )
					  AND ($3::uuid[] IS NULL OR cardinality($3::uuid[]) = 0 OR NOT (q.id = ANY($3::uuid[])))
					ORDER BY RANDOM()
					LIMIT $4
				`, dt.difficulty, jobTags, selectedIDs, neededDiff)
				if err != nil {
					return nil, fmt.Errorf("tier 3 question selection failed for %s: %w", dt.difficulty, err)
				}

				var tier3IDs []uuid.UUID
				for rows.Next() {
					var id uuid.UUID
					if err := rows.Scan(&id); err != nil {
						rows.Close()
						return nil, err
					}
					tier3IDs = append(tier3IDs, id)
				}
				rows.Close()
				selectedByDiff[dt.difficulty] += len(tier3IDs)
				addSelected(tier3IDs)
			}
		}

		// TIER 4: Any active question (fallback if still short)
		if len(selectedIDs) < totalTarget {
			needed := totalTarget - len(selectedIDs)
			rows, err := s.pool.Query(ctx, `
				SELECT q.id FROM questions q
				WHERE q.is_active = true
				  AND (q.question_type != 'coding_challenge' OR EXISTS (
				      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
				  ))
				  AND ($1::uuid[] IS NULL OR cardinality($1::uuid[]) = 0 OR NOT (q.id = ANY($1::uuid[])))
				ORDER BY RANDOM()
				LIMIT $2
			`, selectedIDs, needed)
			if err != nil {
				return nil, fmt.Errorf("tier 4 question selection failed: %w", err)
			}

			var tier4IDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return nil, err
				}
				tier4IDs = append(tier4IDs, id)
			}
			rows.Close()
			addSelected(tier4IDs)
		}
	} else {
		// Job has NO tags: tag restriction is omitted
		// Select by requested difficulty (3 easy, 3 medium, 3 hard, 1 expert)
		for _, dt := range difficultyTargets {
			needed := dt.count
			if needed <= 0 || len(selectedIDs) >= totalTarget {
				continue
			}

			rows, err := s.pool.Query(ctx, `
				SELECT q.id FROM questions q
				WHERE q.is_active = true
				  AND (q.question_type != 'coding_challenge' OR EXISTS (
				      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
				  ))
				  AND q.difficulty = $1
				  AND ($2::uuid[] IS NULL OR cardinality($2::uuid[]) = 0 OR NOT (q.id = ANY($2::uuid[])))
				ORDER BY RANDOM()
				LIMIT $3
			`, dt.difficulty, selectedIDs, needed)
			if err != nil {
				return nil, fmt.Errorf("no-tag difficulty question selection failed for %s: %w", dt.difficulty, err)
			}

			var dtIDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return nil, err
				}
				dtIDs = append(dtIDs, id)
			}
			rows.Close()
			addSelected(dtIDs)
		}

		// Fallback to any active question if requested difficulties fell short
		if len(selectedIDs) < totalTarget {
			needed := totalTarget - len(selectedIDs)
			rows, err := s.pool.Query(ctx, `
				SELECT q.id FROM questions q
				WHERE q.is_active = true
				  AND (q.question_type != 'coding_challenge' OR EXISTS (
				      SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
				  ))
				  AND ($1::uuid[] IS NULL OR cardinality($1::uuid[]) = 0 OR NOT (q.id = ANY($1::uuid[])))
				ORDER BY RANDOM()
				LIMIT $2
			`, selectedIDs, needed)
			if err != nil {
				return nil, fmt.Errorf("no-tag fallback question selection failed: %w", err)
			}

			var fallbackIDs []uuid.UUID
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return nil, err
				}
				fallbackIDs = append(fallbackIDs, id)
			}
			rows.Close()
			addSelected(fallbackIDs)
		}
	}

	if len(selectedIDs) < totalTarget {
		return nil, fmt.Errorf("insufficient questions in question bank to generate a 10-question quiz (found %d, need %d)", len(selectedIDs), totalTarget)
	}

	finalIDs := make([]uuid.UUID, totalTarget)
	copy(finalIDs, selectedIDs[:totalTarget])

	// Randomize/order selected questions ONCE
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(finalIDs), func(i, j int) {
		finalIDs[i], finalIDs[j] = finalIDs[j], finalIDs[i]
	})

	return finalIDs, nil
}

// StartQuizResponse represents the result of starting or resuming a quiz attempt
type StartQuizResponse struct {
	Message          string                 `json:"message"`
	AttemptID        string                 `json:"attempt_id"`
	Status           string                 `json:"status"`
	Question         map[string]interface{} `json:"question,omitempty"`
	QuestionNumber   int                    `json:"question_number,omitempty"`
	TotalQuestions   int                    `json:"total_questions,omitempty"`
	Answered         int                    `json:"answered"`
	Skipped          int                    `json:"skipped"`
	RemainingSeconds int                    `json:"remaining_seconds"`
}

// StartQuizAttempt initializes or resumes an active quiz session in PostgreSQL
func (s *QuizService) StartQuizAttempt(ctx context.Context, attemptID, userID, appID, jobID string) (*StartQuizResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var (
		appUUID        uuid.UUID
		jobUUID        uuid.UUID
		attemptUUID    uuid.UUID
		hasAttemptUUID bool
	)

	if attemptID != "" {
		if parsed, err := uuid.Parse(attemptID); err == nil {
			attemptUUID = parsed
			hasAttemptUUID = true

			// Look up attempt in database to infer appUUID and jobUUID if missing
			var (
				dbAppID  uuid.UUID
				dbUserID uuid.UUID
				dbJobID  uuid.UUID
			)
			err = s.pool.QueryRow(ctx, `
				SELECT application_id, user_id, job_id
				FROM quiz_attempts
				WHERE id = $1
			`, attemptUUID).Scan(&dbAppID, &dbUserID, &dbJobID)
			if err == nil {
				if dbUserID != userUUID {
					return nil, fmt.Errorf("quiz attempt does not belong to this user")
				}
				appUUID = dbAppID
				jobUUID = dbJobID
			}
		}
	}

	if appUUID == uuid.Nil && appID != "" {
		if parsed, err := uuid.Parse(appID); err == nil {
			appUUID = parsed
		}
	}
	if jobUUID == uuid.Nil && jobID != "" {
		if parsed, err := uuid.Parse(jobID); err == nil {
			jobUUID = parsed
		}
	}

	// 1. If application UUID is known, verify ownership and job match
	if appUUID != uuid.Nil {
		var (
			appUserID uuid.UUID
			appJobID  uuid.UUID
			appStatus string
		)
		err = s.pool.QueryRow(ctx, `
			SELECT user_id, job_id, status FROM job_applications WHERE id = $1
		`, appUUID).Scan(&appUserID, &appJobID, &appStatus)
		if err != nil {
			return nil, fmt.Errorf("application not found: %w", err)
		}
		if appUserID != userUUID {
			return nil, fmt.Errorf("application does not belong to this user")
		}
		if jobUUID != uuid.Nil && appJobID != jobUUID {
			return nil, fmt.Errorf("application does not match job")
		}
		if jobUUID == uuid.Nil {
			jobUUID = appJobID
		}
	}

	// 2. Check if an attempt already exists for this ID or application
	var (
		existingID     uuid.UUID
		existingUserID uuid.UUID
		existingStatus string
	)
	if hasAttemptUUID && appUUID != uuid.Nil {
		err = s.pool.QueryRow(ctx, `
			SELECT id, user_id, status FROM quiz_attempts WHERE id = $1 OR application_id = $2
		`, attemptUUID, appUUID).Scan(&existingID, &existingUserID, &existingStatus)
	} else if hasAttemptUUID {
		err = s.pool.QueryRow(ctx, `
			SELECT id, user_id, status FROM quiz_attempts WHERE id = $1
		`, attemptUUID).Scan(&existingID, &existingUserID, &existingStatus)
	} else if appUUID != uuid.Nil {
		err = s.pool.QueryRow(ctx, `
			SELECT id, user_id, status FROM quiz_attempts WHERE application_id = $1
		`, appUUID).Scan(&existingID, &existingUserID, &existingStatus)
	} else {
		return nil, fmt.Errorf("either valid quiz attempt ID or application ID is required")
	}

	if err == nil {
		if existingUserID != userUUID {
			return nil, fmt.Errorf("quiz attempt does not belong to this user")
		}

		// Attempt already exists
		if existingStatus == "completed" {
			return nil, fmt.Errorf("quiz attempt for this application is already completed")
		}
		if existingStatus == "timed_out" || existingStatus == "abandoned" {
			return nil, fmt.Errorf("quiz attempt is closed (status: %s)", existingStatus)
		}

		// Check if existing attempt has exceeded time limit
		var (
			existingStartedAt pgtype.Timestamp
			existingTimeLimit int
		)
		_ = s.pool.QueryRow(ctx, `
			SELECT started_at, COALESCE(time_limit_minutes, 0) FROM quiz_attempts WHERE id = $1
		`, existingID).Scan(&existingStartedAt, &existingTimeLimit)
		if existingTimeLimit > 0 && existingStartedAt.Valid && time.Since(existingStartedAt.Time) > time.Duration(existingTimeLimit)*time.Minute {
			_, _ = s.pool.Exec(ctx, `
				UPDATE quiz_attempts SET status = 'timed_out', completed_at = NOW(), updated_at = NOW() WHERE id = $1
			`, existingID)
			return nil, fmt.Errorf("quiz attempt is closed (status: timed_out)")
		}

		// Ensure status is transitioned to in_progress
		if existingStatus == "started" {
			_, _ = s.pool.Exec(ctx, `
				UPDATE quiz_attempts
				SET status = 'in_progress', last_activity_at = NOW()
				WHERE id = $1 AND status = 'started'
			`, existingID)
			existingStatus = "in_progress"
		}

		// Ensure persisted questions exist for this existing attempt
		var qCount int
		err = s.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM quiz_attempt_questions WHERE quiz_attempt_id = $1
		`, existingID).Scan(&qCount)
		if err == nil && qCount == 0 {
			// Backfill persisted questions if missing
			selectedQuestionIDs, selectErr := s.selectQuizQuestions(ctx, jobUUID)
			if selectErr != nil {
				return nil, selectErr
			}
			for i, qID := range selectedQuestionIDs {
				_, _ = s.pool.Exec(ctx, `
					INSERT INTO quiz_attempt_questions (id, quiz_attempt_id, question_id, question_order, created_at)
					VALUES ($1, $2, $3, $4, NOW())
					ON CONFLICT (quiz_attempt_id, question_order) DO NOTHING
				`, uuid.New(), existingID, qID, i+1)
			}
		}

		// Return/resume the existing attempt
		log.Println("Active attempt already exists, resuming:", existingID)
		currentQ, nextErr := s.GetNextQuestion(ctx, existingID.String(), userID)
		if nextErr != nil {
			return nil, fmt.Errorf("failed to retrieve current question: %w", nextErr)
		}

		qNum := 1
		totalQ := 10
		if n, ok := currentQ["question_number"].(int); ok {
			qNum = n
		}
		if t, ok := currentQ["total_questions"].(int); ok {
			totalQ = t
		}

		ans := 0
		skp := 0
		remSec := 0
		if a, ok := currentQ["answered"].(int); ok {
			ans = a
		}
		if s, ok := currentQ["skipped"].(int); ok {
			skp = s
		}
		if r, ok := currentQ["remaining_seconds"].(int); ok {
			remSec = r
		}

		return &StartQuizResponse{
			Message:          "Quiz resumed successfully",
			AttemptID:        existingID.String(),
			Status:           existingStatus,
			Question:         currentQ,
			QuestionNumber:   qNum,
			TotalQuestions:   totalQ,
			Answered:         ans,
			Skipped:          skp,
			RemainingSeconds: remSec,
		}, nil
	}

	// 3. Select complete question set ONCE
	if jobUUID == uuid.Nil {
		return nil, fmt.Errorf("job ID is required to initialize quiz")
	}
	if appUUID == uuid.Nil {
		return nil, fmt.Errorf("application ID is required to initialize quiz")
	}

	selectedQuestionIDs, err := s.selectQuizQuestions(ctx, jobUUID)
	if err != nil {
		return nil, err
	}

	// 4. Create attempt and persist questions in a transaction
	if !hasAttemptUUID {
		attemptUUID = uuid.New()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO quiz_attempts (
			id, application_id, user_id, job_id, total_questions,
			questions_per_quiz, time_limit_minutes, passing_score, status,
			started_at, last_activity_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 10, 10, 30, 70, 'in_progress', NOW(), NOW(), NOW(), NOW())
	`, attemptUUID, appUUID, userUUID, jobUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create quiz attempt: %w", err)
	}

	for i, qID := range selectedQuestionIDs {
		order := i + 1
		_, err = tx.Exec(ctx, `
			INSERT INTO quiz_attempt_questions (
				id, quiz_attempt_id, question_id, question_order, created_at
			) VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), attemptUUID, qID, order)
		if err != nil {
			return nil, fmt.Errorf("failed to persist quiz attempt question %d: %w", order, err)
		}
	}

	// Update job_applications status to quiz_started and set quiz_id
	_, err = tx.Exec(ctx, `
		UPDATE job_applications
		SET status = 'quiz_started', quiz_id = $2, updated_at = NOW()
		WHERE id = $1
	`, appUUID, attemptUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to update application status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit quiz attempt creation: %w", err)
	}

	// 5. Return attempt and the first question
	firstQ, err := s.GetNextQuestion(ctx, attemptUUID.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve first question: %w", err)
	}

	ans := 0
	skp := 0
	remSec := 0
	if a, ok := firstQ["answered"].(int); ok {
		ans = a
	}
	if s, ok := firstQ["skipped"].(int); ok {
		skp = s
	}
	if r, ok := firstQ["remaining_seconds"].(int); ok {
		remSec = r
	}

	return &StartQuizResponse{
		Message:          "Quiz started successfully",
		AttemptID:        attemptUUID.String(),
		Status:           "in_progress",
		Question:         firstQ,
		QuestionNumber:   1,
		TotalQuestions:   10,
		Answered:         ans,
		Skipped:          skp,
		RemainingSeconds: remSec,
	}, nil
}

// sanitizeQuizOptions strips any answer-revealing fields from question options.
// For multiple-choice questions, only available option identifiers and display text are returned.
func sanitizeQuizOptions(optionsRaw json.RawMessage) json.RawMessage {
	if len(optionsRaw) == 0 || string(optionsRaw) == "null" {
		return json.RawMessage(`[]`)
	}

	rawBytes := []byte(optionsRaw)
	var strVal string
	if err := json.Unmarshal(optionsRaw, &strVal); err == nil {
		rawBytes = []byte(strVal)
	}

	// Try unmarshaling as slice of objects (e.g. [{"option": "A", "text": "...", "is_correct": true}])
	var objList []map[string]interface{}
	if err := json.Unmarshal(rawBytes, &objList); err == nil {
		sanitizedList := make([]map[string]interface{}, len(objList))
		for i, item := range objList {
			cleanItem := make(map[string]interface{})
			for k, v := range item {
				lk := strings.ToLower(k)
				if lk == "is_correct" || lk == "iscorrect" || lk == "correct" ||
					lk == "is_answer" || lk == "isanswer" || lk == "answer" ||
					lk == "correct_answer" || lk == "correctanswer" || lk == "explanation" {
					continue
				}
				cleanItem[k] = v
			}
			sanitizedList[i] = cleanItem
		}
		res, err := json.Marshal(sanitizedList)
		if err == nil {
			return json.RawMessage(res)
		}
	}

	// Try unmarshaling as slice of strings (e.g. ["Option A", "Option B"])
	var strList []string
	if err := json.Unmarshal(rawBytes, &strList); err == nil {
		res, err := json.Marshal(strList)
		if err == nil {
			return json.RawMessage(res)
		}
	}

	return optionsRaw
}

// GetNextQuestion retrieves the next persisted unanswered question for this attempt
func (s *QuizService) GetNextQuestion(ctx context.Context, attemptID string, userID string) (map[string]interface{}, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 1. Fetch attempt and verify ownership
	var (
		qaUserID         uuid.UUID
		qaJobID          uuid.UUID
		status           string
		questionsPerQuiz int
		totalQuestions   int
		timeLimitMinutes int
		startedAt        pgtype.Timestamp
	)
	err = s.pool.QueryRow(ctx, `
		SELECT user_id, job_id, status, questions_per_quiz, total_questions, COALESCE(time_limit_minutes, 0), started_at
		FROM quiz_attempts
		WHERE id = $1
	`, attemptUUID).Scan(&qaUserID, &qaJobID, &status, &questionsPerQuiz, &totalQuestions, &timeLimitMinutes, &startedAt)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt not found: %w", err)
	}
	if qaUserID != userUUID {
		return nil, fmt.Errorf("quiz attempt does not belong to this user")
	}

	limitCount := questionsPerQuiz
	if limitCount <= 0 {
		limitCount = totalQuestions
	}
	if limitCount <= 0 {
		limitCount = 10
	}

	// 2. Calculate lightweight answer counters efficiently (answered and skipped; correctness is kept internal to DB)
	var (
		skippedCount  int
		answeredCount int
	)
	err = s.pool.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE qa.is_skipped = true),
			COUNT(*)
		FROM quiz_answers qa
		WHERE qa.quiz_attempt_id = $1
	`, attemptUUID).Scan(&skippedCount, &answeredCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count answered questions: %w", err)
	}

	// 3. Check status & overall timeout
	if timeLimitMinutes > 0 && startedAt.Valid {
		totalSec := timeLimitMinutes * 60
		elapsed := int(time.Since(startedAt.Time).Seconds())
		if elapsed >= totalSec {
			_, _ = s.pool.Exec(ctx, `
				UPDATE quiz_attempts
				SET status = 'timed_out', completed_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND status IN ('started', 'in_progress', 'paused')
			`, attemptUUID)
			return map[string]interface{}{
				"status":              "finished",
				"message":             "Quiz time limit has expired (timed_out)",
				"attempt_id":          attemptID,
				"question_number":     limitCount,
				"total_questions":     limitCount,
				"answered":            answeredCount,
				"skipped":             skippedCount,
				"remaining_seconds":   0,
				"remaining_questions": 0,
				"is_last_question":    true,
			}, nil
		}
	}

	if status == "completed" {
		return map[string]interface{}{
			"status":              "finished",
			"message":             "You have answered all questions in this quiz",
			"attempt_id":          attemptID,
			"question_number":     limitCount,
			"total_questions":     limitCount,
			"answered":            answeredCount,
			"skipped":             skippedCount,
			"remaining_seconds":   0,
			"remaining_questions": 0,
			"is_last_question":    true,
		}, nil
	}
	if status == "timed_out" || status == "abandoned" {
		return map[string]interface{}{
			"status":              "finished",
			"message":             fmt.Sprintf("Quiz attempt is %s", status),
			"attempt_id":          attemptID,
			"question_number":     limitCount,
			"total_questions":     limitCount,
			"answered":            answeredCount,
			"skipped":             skippedCount,
			"remaining_seconds":   0,
			"remaining_questions": 0,
			"is_last_question":    true,
		}, nil
	}

	// If the attempt has not been started yet (status == 'started'), return ready message so frontend shows the Start Quiz screen
	if status == "started" {
		return map[string]interface{}{
			"status":              "finished",
			"message":             "No more questions available or quiz completed",
			"attempt_id":          attemptID,
			"question_number":     0,
			"total_questions":     limitCount,
			"answered":            0,
			"skipped":             0,
			"remaining_seconds":   0,
			"remaining_questions": limitCount,
			"is_last_question":    false,
		}, nil
	}

	if answeredCount >= limitCount {
		return map[string]interface{}{
			"status":              "finished",
			"message":             "You have answered all questions in this quiz",
			"attempt_id":          attemptID,
			"question_number":     limitCount,
			"total_questions":     limitCount,
			"answered":            answeredCount,
			"skipped":             skippedCount,
			"remaining_seconds":   0,
			"remaining_questions": 0,
			"is_last_question":    true,
		}, nil
	}

	// 4. Ensure persistent sequence exists in quiz_attempt_questions (backfill on the fly if 0)
	var qaqCount int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quiz_attempt_questions WHERE quiz_attempt_id = $1
	`, attemptUUID).Scan(&qaqCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count attempt questions: %w", err)
	}

	if qaqCount == 0 {
		selectedQuestionIDs, selectErr := s.selectQuizQuestions(ctx, qaJobID)
		if selectErr != nil {
			log.Printf("GetNextQuestion: failed to select questions for attempt %s: %v", attemptUUID, selectErr)
			return nil, fmt.Errorf("failed to select quiz questions: %w", selectErr)
		}
		for i, qID := range selectedQuestionIDs {
			_, insErr := s.pool.Exec(ctx, `
				INSERT INTO quiz_attempt_questions (id, quiz_attempt_id, question_id, question_order, created_at)
				VALUES ($1, $2, $3, $4, NOW())
				ON CONFLICT (quiz_attempt_id, question_order) DO NOTHING
			`, uuid.New(), attemptUUID, qID, i+1)
			if insErr != nil {
				log.Printf("GetNextQuestion: failed to insert quiz_attempt_question %d: %v", i+1, insErr)
			}
		}
		qaqCount = len(selectedQuestionIDs)
	}

	if qaqCount == 0 {
		return map[string]interface{}{
			"status":              "finished",
			"message":             "No more questions available or quiz completed",
			"attempt_id":          attemptID,
			"question_number":     limitCount,
			"total_questions":     limitCount,
			"answered":            answeredCount,
			"skipped":             skippedCount,
			"remaining_seconds":   0,
			"remaining_questions": 0,
			"is_last_question":    true,
		}, nil
	}

	// 5. Retrieve next unanswered question from persisted quiz_attempt_questions.
	// NOTE: correct_answer is kept strictly server-side and never returned during an active quiz.
	var (
		questionOrder    int
		id               uuid.UUID
		qText, qType     string
		difficulty       string
		options          *string
		timeLimitSeconds int
	)
	err = s.pool.QueryRow(ctx, `
		SELECT 
			qaq.question_order,
			q.id,
			q.question_text,
			q.question_type::text,
			q.options::text,
			q.difficulty::text,
			q.time_limit_seconds
		FROM quiz_attempt_questions qaq
		JOIN questions q ON q.id = qaq.question_id
		WHERE qaq.quiz_attempt_id = $1
		  AND NOT EXISTS (
		      SELECT 1 FROM quiz_answers qa
		      WHERE qa.quiz_attempt_id = qaq.quiz_attempt_id
		        AND qa.question_id = qaq.question_id
		  )
		ORDER BY qaq.question_order ASC
		LIMIT 1
	`, attemptUUID).Scan(&questionOrder, &id, &qText, &qType, &options, &difficulty, &timeLimitSeconds)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return map[string]interface{}{
				"status":              "finished",
				"message":             "You have answered all questions in this quiz",
				"attempt_id":          attemptID,
				"question_number":     limitCount,
				"total_questions":     limitCount,
				"answered":            answeredCount,
				"skipped":             skippedCount,
				"remaining_seconds":   0,
				"remaining_questions": 0,
				"is_last_question":    true,
			}, nil
		}
		log.Printf("GetNextQuestion query error for attempt %s: %v", attemptID, err)
		return nil, fmt.Errorf("failed to retrieve next question: %w", err)
	}

	var optionsRaw json.RawMessage
	if options != nil {
		optionsRaw = sanitizeQuizOptions(json.RawMessage(*options))
	} else {
		optionsRaw = json.RawMessage(`[]`)
	}

	var codingDetails map[string]interface{}
	// For coding_challenge, also fetch coding_details (without hidden test cases)
	if qType == "coding_challenge" {
		var lang string
		var codeTemplate *string
		var testCases json.RawMessage
		var execTimeLimit, memLimit int

		err := s.pool.QueryRow(ctx, `
			SELECT language, code_template, test_cases, execution_time_limit, memory_limit
			FROM coding_questions
			WHERE question_id = $1
		`, id).Scan(&lang, &codeTemplate, &testCases, &execTimeLimit, &memLimit)
		if err != nil {
			log.Printf("GetNextQuestion: no coding_details for question %s: %v", id, err)
		} else {
			var allTests []map[string]interface{}
			var visibleTests []map[string]interface{}
			if err := json.Unmarshal(testCases, &allTests); err == nil {
				for _, tc := range allTests {
					if hidden, _ := tc["hidden"].(bool); hidden {
						continue
					}
					if hidden, _ := tc["is_hidden"].(bool); hidden {
						continue
					}
					visibleTests = append(visibleTests, tc)
				}
			}
			filteredTests, _ := json.Marshal(visibleTests)

			codingDetails = map[string]interface{}{
				"language":             lang,
				"code_template":        codeTemplate,
				"test_cases":           json.RawMessage(filteredTests),
				"execution_time_limit": execTimeLimit,
				"memory_limit":         memLimit,
			}
		}
	}

	remaining := limitCount - answeredCount
	isLast := questionOrder >= limitCount

	remainingSeconds := timeLimitSeconds
	if timeLimitMinutes > 0 && startedAt.Valid {
		totalSec := timeLimitMinutes * 60
		elapsed := int(time.Since(startedAt.Time).Seconds())
		rem := totalSec - elapsed
		if rem < 0 {
			rem = 0
		}
		remainingSeconds = rem
	} else if timeLimitSeconds > 0 {
		remainingSeconds = timeLimitSeconds
	}

	qObj := map[string]interface{}{
		"id":                 id.String(),
		"text":               qText,
		"question_text":      qText,
		"type":               qType,
		"question_type":      qType,
		"difficulty":         difficulty,
		"time_limit_seconds": timeLimitSeconds,
		"options":            optionsRaw,
		"status":             "unanswered",
	}
	if codingDetails != nil {
		qObj["coding_details"] = codingDetails
	}

	resp := map[string]interface{}{
		"question_number":     questionOrder,
		"total_questions":     limitCount,
		"answered":            answeredCount,
		"skipped":             skippedCount,
		"remaining_seconds":   remainingSeconds,
		"remaining_questions": remaining,
		"is_last_question":    isLast,
		"status":              status,
		"attempt_id":          attemptID,
		"id":                  id.String(),
		"question_text":       qText,
		"question_type":       qType,
		"difficulty":          difficulty,
		"time_limit_seconds":  timeLimitSeconds,
		"options":             optionsRaw,
		"question":            qObj,
	}
	if codingDetails != nil {
		resp["coding_details"] = codingDetails
	}
	// Update last_activity_at to record the exact time the question was dispatched to user
	_, _ = s.pool.Exec(ctx, `UPDATE quiz_attempts SET last_activity_at = NOW() WHERE id = $1`, attemptUUID)

	return resp, nil
}

// SaveQuizAnswer updates the database state by cataloging the submitted single question response
func (s *QuizService) SaveQuizAnswer(ctx context.Context, attemptID, userID, questionID, answer string, timeSpent int, isSkipped bool) error {
	log.Printf("DEBUG: Validating and saving answer for AttemptID: %s, QuestionID: %s, UserID: %s", attemptID, questionID, userID)

	// 1. Convert strings to UUIDs
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	questionUUID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("invalid question ID: %w", err)
	}

	// 2. Validate attempt existence, user ownership, and state inside a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		qaUserID         uuid.UUID
		status           string
		startedAt        pgtype.Timestamp
		timeLimitMinutes int
		lastActivityAt   pgtype.Timestamp
	)
	err = tx.QueryRow(ctx, `
		SELECT user_id, status, started_at, COALESCE(time_limit_minutes, 0), last_activity_at
		FROM quiz_attempts WHERE id = $1 FOR UPDATE
	`, attemptUUID).Scan(&qaUserID, &status, &startedAt, &timeLimitMinutes, &lastActivityAt)
	if err != nil {
		return fmt.Errorf("quiz attempt not found: %w", err)
	}
	if qaUserID != userUUID {
		return fmt.Errorf("quiz attempt does not belong to this user")
	}
	if status == "completed" {
		return fmt.Errorf("cannot answer questions for completed quiz attempt")
	}
	if status == "timed_out" || status == "abandoned" {
		return fmt.Errorf("quiz attempt is no longer active (status: %s)", status)
	}
	if status != "started" && status != "in_progress" && status != "paused" {
		return fmt.Errorf("quiz attempt is no longer active (status: %s)", status)
	}

	// Check overall quiz time limit
	if timeLimitMinutes > 0 && startedAt.Valid {
		totalSec := timeLimitMinutes * 60
		elapsed := int(time.Since(startedAt.Time).Seconds())
		if elapsed > totalSec+15 { // 15s grace for network/clock skew
			_, _ = tx.Exec(ctx, `
				UPDATE quiz_attempts
				SET status = 'timed_out', completed_at = NOW(), updated_at = NOW()
				WHERE id = $1
			`, attemptUUID)
			_ = tx.Commit(ctx)
			return fmt.Errorf("quiz attempt has timed out")
		}
	}

	// 3. Verify question belongs to that attempt through quiz_attempt_questions
	var submittedOrder int
	err = tx.QueryRow(ctx, `
		SELECT question_order FROM quiz_attempt_questions
		WHERE quiz_attempt_id = $1 AND question_id = $2
	`, attemptUUID, questionUUID).Scan(&submittedOrder)
	if err != nil {
		return fmt.Errorf("question does not belong to this quiz attempt")
	}

	// 4. Verify question sequence (allows updating already answered questions; prevents skipping ahead)
	var alreadyAnswered bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM quiz_answers WHERE quiz_attempt_id = $1 AND question_id = $2
		)
	`, attemptUUID, questionUUID).Scan(&alreadyAnswered)
	if err != nil {
		return fmt.Errorf("failed to check existing answer: %w", err)
	}

	if !alreadyAnswered {
		var expectedOrder int
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(MIN(qaq.question_order), 0)::int
			FROM quiz_attempt_questions qaq
			WHERE qaq.quiz_attempt_id = $1
			  AND NOT EXISTS (
			      SELECT 1 FROM quiz_answers qa
			      WHERE qa.quiz_attempt_id = qaq.quiz_attempt_id
			        AND qa.question_id = qaq.question_id
			  )
		`, attemptUUID).Scan(&expectedOrder)
		if err != nil {
			return fmt.Errorf("failed to determine expected question order: %w", err)
		}

		if expectedOrder == 0 {
			return fmt.Errorf("all questions for this quiz attempt have already been answered")
		}

		if submittedOrder > expectedOrder {
			return fmt.Errorf("cannot answer questions out of order: expected question %d, got %d", expectedOrder, submittedOrder)
		}
	}

	// 5. Fetch question info from questions table
	var (
		dbCorrectAnswer  *string
		timeLimitSeconds int
		questionType     string
	)
	err = tx.QueryRow(ctx, `
		SELECT correct_answer, time_limit_seconds, question_type::text
		FROM questions WHERE id = $1
	`, questionUUID).Scan(&dbCorrectAnswer, &timeLimitSeconds, &questionType)
	if err != nil {
		return fmt.Errorf("failed to fetch question details: %w", err)
	}

	// 6. Enforce time limit with server-authoritative verification
	if timeLimitSeconds > 0 {
		if timeSpent > timeLimitSeconds {
			return fmt.Errorf("time limit exceeded for this question (%d seconds)", timeLimitSeconds)
		}
		if !alreadyAnswered && lastActivityAt.Valid {
			serverElapsed := int(time.Since(lastActivityAt.Time).Seconds())
			if serverElapsed > timeLimitSeconds+15 {
				return fmt.Errorf("time limit exceeded for this question (%d seconds)", timeLimitSeconds)
			}
		}
	}
	if timeSpent < 0 {
		timeSpent = 0
	}

	// 7. Determine correctness
	isCodingQuestion := questionType == "coding_challenge"
	isCorrect := false
	if isCodingQuestion {
		var existingCorrect *bool
		_ = tx.QueryRow(ctx,
			"SELECT is_correct FROM quiz_answers WHERE quiz_attempt_id = $1 AND question_id = $2",
			attemptUUID, questionUUID).Scan(&existingCorrect)
		if existingCorrect != nil {
			isCorrect = *existingCorrect
		}
	} else {
		isCorrect = dbCorrectAnswer != nil && answer == *dbCorrectAnswer
	}

	// 8. Insert or update quiz_answers
	newAnswerID := uuid.New()
	var savedAnswerID uuid.UUID
	query := `
		INSERT INTO quiz_answers (
			id, quiz_attempt_id, question_id, user_answer, time_spent_seconds, is_skipped, is_correct, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (quiz_attempt_id, question_id)
		DO UPDATE SET
			user_answer = EXCLUDED.user_answer,
			is_correct = EXCLUDED.is_correct,
			time_spent_seconds = quiz_answers.time_spent_seconds + EXCLUDED.time_spent_seconds,
			is_skipped = EXCLUDED.is_skipped,
			save_count = quiz_answers.save_count + 1,
			last_saved_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`
	err = tx.QueryRow(ctx, query, newAnswerID, attemptUUID, questionUUID, answer, timeSpent, isSkipped, isCorrect).Scan(&savedAnswerID)
	if err != nil {
		return fmt.Errorf("failed to save quiz answer: %w", err)
	}

	// 9. quiz_answer_history writes stopped to prevent O(attempts * questions * saves) bloat.
	// Historical data retained; writes can be re-enabled if dedicated audit requirements exist.
	_ = savedAnswerID

	// 10. Update last_activity_at and transition status to in_progress if started
	_, err = tx.Exec(ctx, `
		UPDATE quiz_attempts
		SET last_activity_at = NOW(),
		    time_spent_seconds = time_spent_seconds + $2,
		    status = CASE WHEN status = 'started' THEN 'in_progress'::quiz_attempt_status ELSE status END
		WHERE id = $1
	`, attemptUUID, timeSpent)
	if err != nil {
		return fmt.Errorf("failed to update quiz attempt: %w", err)
	}

	return tx.Commit(ctx)
}

// RunQuizCode executes user code against visible test cases for a coding_challenge question
func (s *QuizService) RunQuizCode(ctx context.Context, attemptID, userID, questionID, language, code string) (*models.ExecuteResponse, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	questionUUID, err := uuid.Parse(questionID)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %w", err)
	}

	// Verify attempt ownership, active status, and time limit
	var (
		qaUserID         uuid.UUID
		status           string
		startedAt        pgtype.Timestamp
		timeLimitMinutes int
	)
	err = s.pool.QueryRow(ctx, `
		SELECT user_id, status, started_at, COALESCE(time_limit_minutes, 0) FROM quiz_attempts WHERE id = $1
	`, attemptUUID).Scan(&qaUserID, &status, &startedAt, &timeLimitMinutes)
	if err != nil {
		return nil, fmt.Errorf("quiz attempt not found: %w", err)
	}
	if qaUserID != userUUID {
		return nil, fmt.Errorf("quiz attempt does not belong to this user")
	}
	if status == "completed" {
		return nil, fmt.Errorf("cannot run code for completed quiz attempt")
	}
	if status == "timed_out" || status == "abandoned" {
		return nil, fmt.Errorf("quiz attempt is no longer active (status: %s)", status)
	}
	if status != "started" && status != "in_progress" && status != "paused" {
		return nil, fmt.Errorf("quiz attempt is no longer active (status: %s)", status)
	}

	// Check overall quiz time limit
	if timeLimitMinutes > 0 && startedAt.Valid {
		totalSec := timeLimitMinutes * 60
		elapsed := int(time.Since(startedAt.Time).Seconds())
		if elapsed > totalSec+15 {
			_, _ = s.pool.Exec(ctx, `
				UPDATE quiz_attempts
				SET status = 'timed_out', completed_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND status IN ('started', 'in_progress', 'paused')
			`, attemptUUID)
			return nil, fmt.Errorf("quiz attempt has timed out")
		}
	}

	// Verify question belongs to that attempt through quiz_attempt_questions
	var submittedOrder int
	err = s.pool.QueryRow(ctx, `
		SELECT question_order FROM quiz_attempt_questions
		WHERE quiz_attempt_id = $1 AND question_id = $2
	`, attemptUUID, questionUUID).Scan(&submittedOrder)
	if err != nil {
		return nil, fmt.Errorf("question does not belong to this quiz attempt")
	}

	// Prevent running code for future questions out of sequence
	var alreadyAnswered bool
	_ = s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM quiz_answers WHERE quiz_attempt_id = $1 AND question_id = $2)
	`, attemptUUID, questionUUID).Scan(&alreadyAnswered)

	if !alreadyAnswered {
		var expectedOrder int
		_ = s.pool.QueryRow(ctx, `
			SELECT COALESCE(MIN(qaq.question_order), 0)::int
			FROM quiz_attempt_questions qaq
			WHERE qaq.quiz_attempt_id = $1
			  AND NOT EXISTS (
			      SELECT 1 FROM quiz_answers qa
			      WHERE qa.quiz_attempt_id = qaq.quiz_attempt_id
			        AND qa.question_id = qaq.question_id
			  )
		`, attemptUUID).Scan(&expectedOrder)

		if expectedOrder > 0 && submittedOrder > expectedOrder {
			return nil, fmt.Errorf("cannot run code for question out of order: expected question %d, got %d", expectedOrder, submittedOrder)
		}
	}

	// First verify the question exists and is a coding_challenge
	var qType string
	err = s.pool.QueryRow(ctx, `SELECT question_type FROM questions WHERE id = $1`, questionUUID).Scan(&qType)
	if err != nil {
		return nil, fmt.Errorf("question not found: %w", err)
	}
	if qType != "coding_challenge" {
		return nil, fmt.Errorf("question type is %q, not coding_challenge", qType)
	}

	var dbLang string
	var testCases json.RawMessage
	var execTimeLimit, memLimit int

	err = s.pool.QueryRow(ctx, `
		SELECT language, test_cases, execution_time_limit, memory_limit
		FROM coding_questions
		WHERE question_id = $1
	`, questionUUID).Scan(&dbLang, &testCases, &execTimeLimit, &memLimit)
	if err != nil {
		return nil, fmt.Errorf("coding_challenge data not found: the question exists but has no test cases configured in the coding_questions table (missing row for question_id=%s)", questionID)
	}

	// Use provided language or fall back to DB language
	lang := language
	if lang == "" {
		lang = dbLang
	}

	isSQL := strings.EqualFold(lang, "sql") || strings.EqualFold(lang, "sqlite")

	// SQL questions carry their own document shape (imported database +
	// query/verify tests), so normalize them up-front.
	var sqlFiles map[string]string
	var sqlTestsPayload string
	if isSQL {
		payload, files, pErr := BuildSQLPayload(testCases)
		if pErr != nil {
			return nil, fmt.Errorf("invalid SQL test cases: %w", pErr)
		}
		sqlTestsPayload = payload
		sqlFiles = files
	}

	sandbox := &SandboxService{}

	var detectedFunc string
	if !isSQL {
		parseReq := models.ParseRequest{Language: lang, Code: code}
		parseResp, parseErr := sandbox.ParseCode(ctx, parseReq)

		if parseErr == nil && len(parseResp.Functions) > 0 {
			detectedFunc = parseResp.Functions[0].Name
		} else {
			detectedFunc = detectFunctionName(code, lang)
		}
	}

	// Filter to only visible test cases and convert to sandbox format
	var allTests []map[string]interface{}
	if !isSQL {
		if err := json.Unmarshal(testCases, &allTests); err != nil {
			return nil, fmt.Errorf("invalid test cases: %w", err)
		}
	}

	var sandboxTests []map[string]interface{}
	for i, tc := range allTests {
		if hidden, _ := tc["hidden"].(bool); hidden {
			continue
		}
		if hidden, _ := tc["is_hidden"].(bool); hidden {
			continue
		}
		// Convert test case formats
		fn, _ := tc["func"].(string)
		if fn == "" {
			fn = detectedFunc
		} else {
			// Map the expected function name to the user's function name
			fn = detectedFunc
		}
		args := tc["args"]
		if args == nil {
			if input, ok := tc["input"]; ok {
				args = input
			}
		}
		// Validate args: must be an array (list of arguments)
		if args == nil {
			log.Printf("RunQuizCode: skipping test case %d - no args provided", i)
			continue
		}
		if argsStr, ok := args.(string); ok {
			// The entire string is the argument value — parse as JSON and wrap as
			// a SINGLE argument so that e.g. "[1,2,3,4]" becomes [[1,2,3,4]]
			// (one argument: the array) rather than [1,2,3,4] (four arguments).
			var parsed interface{}
			if err := json.Unmarshal([]byte(argsStr), &parsed); err == nil {
				args = []interface{}{parsed}
			} else {
				// Single string value — wrap in array as a single argument
				args = []interface{}{argsStr}
			}
		}
		if _, ok := args.([]interface{}); !ok {
			log.Printf("RunQuizCode: skipping test case %d - args is not an array: %v", i, args)
			continue
		}
		expected := tc["expected"]
		if expected == nil {
			expected = tc["expected_output"]
		}
		if expected == nil {
			expected = tc["output"]
		}
		sandboxTests = append(sandboxTests, map[string]interface{}{
			"func":     fn,
			"args":     args,
			"expected": expected,
		})
	}

	sandboxTestsJSON, _ := json.Marshal(sandboxTests)

	// Detect if test cases are simple input/output (no func/args) — use standard execution
	hasFuncField := false
	for _, tc := range allTests {
		if _, ok := tc["func"]; ok {
			hasFuncField = true
			break
		}
	}

	codeDefinesFunc := hasFunctionDefinition(code, lang)
	useFunctionMode := hasFuncField || codeDefinesFunc || isSQL

	if !useFunctionMode && len(sandboxTests) > 0 {
		// Simple output-based question: run code with stdin, compare stdout to expected
		inputStr, _ := json.Marshal(allTests[0]["input"])
		var stdinVal string
		if inputStr != nil && string(inputStr) != "null" {
			stdinVal = strings.Trim(string(inputStr), "\"")
		}

		req := models.ExecuteRequest{
			Language:    lang,
			Code:        code,
			Type:        models.ExecutionTypeStandard,
			Stdin:       stdinVal,
			TimeLimit:   execTimeLimit,
			MemoryLimit: memLimit,
		}
		resp, err := sandbox.Execute(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("sandbox execution failed: %w", err)
		}

		// Compare stdout to expected output
		if resp.ExitCode == 0 && resp.Stdout != "" {
			expectedOut, _ := allTests[0]["expected_output"].(string)
			actualOut := strings.TrimSpace(resp.Stdout)
			expectedOut = strings.TrimSpace(expectedOut)
			passed := actualOut == expectedOut
			resp.Passed = &passed
		} else {
			passed := false
			resp.Passed = &passed
		}

		isCorrect := resp.Passed != nil && *resp.Passed
		codeOutput := capCodeOutput(resp.Stdout)
		_, upsertErr := s.pool.Exec(ctx, `
			INSERT INTO quiz_answers (quiz_attempt_id, question_id, user_answer, is_correct, code_output, execution_time_ms, save_count, last_saved_at)
			VALUES ($1, $2, $3, $4, $5, $6, 1, NOW())
			ON CONFLICT (quiz_attempt_id, question_id) DO UPDATE SET
				user_answer = EXCLUDED.user_answer,
				is_correct = EXCLUDED.is_correct,
				code_output = EXCLUDED.code_output,
				execution_time_ms = EXCLUDED.execution_time_ms,
				save_count = quiz_answers.save_count + 1,
				last_saved_at = NOW(),
				updated_at = NOW()
		`, attemptID, questionUUID, code, isCorrect, codeOutput, resp.TimeMs)
		if upsertErr != nil {
			log.Printf("RunQuizCode: failed to save quiz_answer: %v", upsertErr)
		}

		return resp, nil
	}

	// Function-based question: use test harness
	req := models.ExecuteRequest{
		Language:    lang,
		Code:        code,
		Type:        models.ExecutionTypeFunction,
		Stdin:       string(sandboxTestsJSON),
		TimeLimit:   execTimeLimit,
		MemoryLimit: memLimit,
	}
	if isSQL {
		req.Stdin = sqlTestsPayload
		req.Files = sqlFiles
	}

	resp, err := sandbox.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("sandbox execution failed: %w", err)
	}

	// Save result to quiz_answers
	isCorrect := resp.Passed != nil && *resp.Passed
	codeOutput := capCodeOutput(resp.Stdout)
	_, upsertErr := s.pool.Exec(ctx, `
		INSERT INTO quiz_answers (quiz_attempt_id, question_id, user_answer, is_correct, code_output, execution_time_ms, save_count, last_saved_at)
		VALUES ($1, $2, $3, $4, $5, $6, 1, NOW())
		ON CONFLICT (quiz_attempt_id, question_id) DO UPDATE SET
			user_answer = EXCLUDED.user_answer,
			is_correct = EXCLUDED.is_correct,
			code_output = EXCLUDED.code_output,
			execution_time_ms = EXCLUDED.execution_time_ms,
			save_count = quiz_answers.save_count + 1,
			last_saved_at = NOW(),
			updated_at = NOW()
	`, attemptID, questionUUID, code, isCorrect, codeOutput, resp.TimeMs)
	if upsertErr != nil {
		log.Printf("RunQuizCode: failed to save quiz_answer: %v", upsertErr)
	}

	return resp, nil
}

const maxCodeOutputBytes = 16 * 1024 // 16 KB hard limit for stored sandbox output

// capCodeOutput truncates stdout to maxCodeOutputBytes to prevent database bloat
func capCodeOutput(s string) *string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > maxCodeOutputBytes {
		trimmed = trimmed[:maxCodeOutputBytes] + "\n...[output truncated: exceeded 16KB limit]"
	}
	return &trimmed
}

// SubmitQuizAttempt calculates and locks progress finality status safely
func (s *QuizService) SubmitQuizAttempt(ctx context.Context, attemptID string, userID string, tags []string) error {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Start transaction and acquire row lock on quiz_attempt
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check attempt existence, ownership, timeout, and current status with FOR UPDATE
	var (
		appID            uuid.UUID
		qaUserID         uuid.UUID
		status           string
		passingScore     int
		questionsPerQuiz int
		startedAt        pgtype.Timestamp
		timeLimitMinutes int
	)
	err = tx.QueryRow(ctx, `
		SELECT application_id, user_id, status, passing_score, questions_per_quiz, started_at, COALESCE(time_limit_minutes, 0)
		FROM quiz_attempts WHERE id = $1 FOR UPDATE
	`, attemptUUID).Scan(&appID, &qaUserID, &status, &passingScore, &questionsPerQuiz, &startedAt, &timeLimitMinutes)
	if err != nil {
		return fmt.Errorf("could not find quiz attempt: %w", err)
	}
	if qaUserID != userUUID {
		return fmt.Errorf("quiz attempt does not belong to this user")
	}
	if status == "completed" {
		return fmt.Errorf("quiz attempt %s is already completed", attemptID)
	}
	if status == "timed_out" || status == "abandoned" {
		return fmt.Errorf("quiz attempt is %s and cannot be submitted", status)
	}
	if status != "started" && status != "in_progress" && status != "paused" {
		return fmt.Errorf("quiz attempt is no longer active (status: %s)", status)
	}

	// Check overall quiz time limit
	if timeLimitMinutes > 0 && startedAt.Valid {
		totalSec := timeLimitMinutes * 60
		elapsed := int(time.Since(startedAt.Time).Seconds())
		if elapsed > totalSec+15 {
			_, _ = tx.Exec(ctx, `
				UPDATE quiz_attempts
				SET status = 'timed_out', completed_at = NOW(), updated_at = NOW()
				WHERE id = $1
			`, attemptUUID)
			_ = tx.Commit(ctx)
			return fmt.Errorf("quiz attempt has timed out and cannot be submitted")
		}
	}

	// Calculate score from persisted answers in quiz_answers for attempt questions
	var (
		correctCount   int
		incorrectCount int
		skippedCount   int
		totalAnswers   int
	)
	err = tx.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE qa.is_correct = true),
			COUNT(*) FILTER (WHERE qa.is_correct = false AND qa.is_skipped = false),
			COUNT(*) FILTER (WHERE qa.is_skipped = true),
			COUNT(*)
		FROM quiz_answers qa
		JOIN quiz_attempt_questions qaq ON qaq.quiz_attempt_id = qa.quiz_attempt_id AND qaq.question_id = qa.question_id
		WHERE qa.quiz_attempt_id = $1
	`, attemptUUID).Scan(&correctCount, &incorrectCount, &skippedCount, &totalAnswers)
	if err != nil {
		return err
	}

	var attemptQuestionCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM quiz_attempt_questions WHERE quiz_attempt_id = $1
	`, attemptUUID).Scan(&attemptQuestionCount)
	if err != nil {
		return err
	}

	totalDenominator := attemptQuestionCount
	if totalDenominator <= 0 {
		totalDenominator = questionsPerQuiz
	}
	if totalDenominator <= 0 {
		totalDenominator = 10
	}

	score := 0.0
	if totalDenominator > 0 {
		score = (float64(correctCount) / float64(totalDenominator)) * 100
	}

	passed := int(score) >= passingScore

	// Update Quiz status to completed atomically
	res, err := tx.Exec(ctx, `
		UPDATE quiz_attempts
		SET status = 'completed',
		    score = $2,
		    correct_answers = $3,
		    incorrect_answers = $4,
		    skipped_questions = $5,
		    passed = $6,
		    completed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1 AND status IN ('started', 'in_progress', 'paused')
	`, attemptUUID, int(score), correctCount, incorrectCount, skippedCount, passed)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("quiz attempt was already completed or not in submittable state")
	}

	// Update the job_applications table
	_, err = tx.Exec(ctx, `
		UPDATE job_applications 
		SET status = 'quiz_completed', 
		    quiz_score = $1, 
		    quiz_completed_at = NOW(),
		    quiz_passed = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, int(score), passed, appID)
	if err != nil {
		return fmt.Errorf("failed to update job_applications table: %w", err)
	}

	return tx.Commit(ctx)
}

func int4Ptr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	return &v.Int32
}

func boolPtr(v pgtype.Bool) *bool {
	if !v.Valid {
		return nil
	}
	return &v.Bool
}

func timestampPtr(v pgtype.Timestamp) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func hasFunctionDefinition(code, lang string) bool {
	code = strings.TrimSpace(code)
	switch strings.ToLower(lang) {
	case "javascript":
		return strings.Contains(code, "function ") ||
			strings.Contains(code, "=>") ||
			strings.Contains(code, "const solution") ||
			strings.Contains(code, "let solution") ||
			strings.Contains(code, "var solution")
	case "python":
		return strings.Contains(code, "def ")
	case "java":
		return strings.Contains(code, "public static") || strings.Contains(code, "class ")
	case "cpp", "c++":
		return strings.Contains(code, "int solution") || strings.Contains(code, "void solution") || strings.Contains(code, "string solution")
	case "go":
		return strings.Contains(code, "func ")
	default:
		return strings.Contains(code, "function ") || strings.Contains(code, "def ") || strings.Contains(code, "func ")
	}
}

func detectFunctionName(code, lang string) string {
	code = strings.TrimSpace(code)
	switch strings.ToLower(lang) {
	case "python":
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "def ") {
				rest := line[4:]
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					return strings.TrimSpace(rest[:parenIdx])
				}
			}
		}
	case "javascript":
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "function ") {
				rest := line[9:]
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					return strings.TrimSpace(rest[:parenIdx])
				}
			}
			if strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") || strings.HasPrefix(line, "var ") {
				equalsIdx := strings.Index(line, "=")
				if equalsIdx > 0 {
					name := strings.TrimSpace(line[6:equalsIdx])
					if name != "" {
						return name
					}
				}
			}
		}
	case "go":
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "func ") {
				rest := line[5:]
				if strings.HasPrefix(rest, "(") {
					closeIdx := strings.Index(rest, ")")
					if closeIdx > 0 {
						rest = strings.TrimSpace(rest[closeIdx+1:])
					}
				}
				parenIdx := strings.Index(rest, "(")
				if parenIdx > 0 {
					name := strings.TrimSpace(rest[:parenIdx])
					if name != "" && name != "main" && name != "init" {
						return name
					}
				}
			}
		}
	case "java":
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "public static") || strings.Contains(line, "static public") {
				parenIdx := strings.Index(line, "(")
				if parenIdx > 0 {
					before := line[:parenIdx]
					parts := strings.Fields(before)
					if len(parts) > 0 {
						name := parts[len(parts)-1]
						if name != "" {
							return name
						}
					}
				}
			}
		}
	case "cpp", "c++":
		for _, line := range strings.Split(code, "\n") {
			line = strings.TrimSpace(line)
			parenIdx := strings.Index(line, "(")
			if parenIdx > 0 {
				before := line[:parenIdx]
				parts := strings.Fields(before)
				if len(parts) >= 2 {
					name := parts[len(parts)-1]
					if name != "main" && name != "if" && name != "for" && name != "while" {
						return name
					}
				}
			}
		}
	}

	for _, line := range strings.Split(code, "\n") {
		line = strings.TrimSpace(line)
		parenIdx := strings.Index(line, "(")
		if parenIdx > 0 {
			before := line[:parenIdx]
			parts := strings.Fields(before)
			if len(parts) > 0 {
				name := parts[len(parts)-1]
				if name != "" && name != "if" && name != "for" && name != "while" && name != "switch" {
					return name
				}
			}
		}
	}

	return "solution"
}
