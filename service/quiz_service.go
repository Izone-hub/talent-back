package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // Tags for Gemini microservice (comma-separated)
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
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
func (s *QuizService) GetJobQuizAttempts(ctx context.Context, jobID string) ([]JobQuizAttempt, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobUUID[:])
	pgJobID.Valid = true

	rows, err := s.queries.ListQuizAttemptsByJob(ctx, pgJobID)
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
func (s *QuizService) GetUserQuizAttempts(ctx context.Context, userID string) ([]JobQuizAttempt, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], userUUID[:])
	pgUserID.Valid = true

	rows, err := s.queries.ListQuizAttemptsByUser(ctx, pgUserID)
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

// QuizReviewAnswer is a single answered question within a quiz attempt review.
type QuizReviewAnswer struct {
	QuestionID       uuid.UUID       `json:"question_id"`
	QuestionText     string          `json:"question_text"`
	QuestionType     string          `json:"question_type"`
	Difficulty       string          `json:"difficulty"`
	Options          json.RawMessage `json:"options"`
	UserAnswer       *string         `json:"user_answer"`
	CorrectAnswer    *string         `json:"correct_answer"`
	IsCorrect        *bool           `json:"is_correct"`
	IsSkipped        bool            `json:"is_skipped"`
	Explanation      *string         `json:"explanation"`
	TimeSpentSeconds int32           `json:"time_spent_seconds"`
	CodeOutput       *string         `json:"code_output"`
	CreatedAt        time.Time       `json:"created_at"`
}

// QuizReview is the full question-by-question view of a quiz attempt,
// including every question the applicant took for the job.
type QuizReview struct {
	ID                uuid.UUID         `json:"id"`
	ApplicationID     uuid.UUID         `json:"application_id"`
	JobApplicationID  uuid.UUID         `json:"job_application_id"`
	UserID            uuid.UUID         `json:"user_id"`
	JobID             uuid.UUID         `json:"job_id"`
	JobTitle          string            `json:"job_title,omitempty"`
	JobCompany        string            `json:"job_company,omitempty"`
	Title             string            `json:"title"`
	Status            string            `json:"status"`
	Score             *int32            `json:"score"`
	Passed            *bool             `json:"passed"`
	CorrectAnswers    *int32            `json:"correct_answers"`
	TotalQuestions    int32             `json:"total_questions"`
	AnsweredQuestions int32             `json:"answered_questions"`
	PassingScore      int32             `json:"passing_score"`
	StartedAt         *time.Time        `json:"started_at"`
	CompletedAt       *time.Time        `json:"completed_at"`
	TimeSpentSeconds  *int32            `json:"time_spent_seconds"`
	Answers           []QuizReviewAnswer `json:"answers"`
}

// GetQuizReview returns the full question-by-question review of a quiz
// attempt. The attempt owner can always view their own attempt; admins can
// view any attempt.
func (s *QuizService) GetQuizReview(ctx context.Context, attemptID, requesterID string, isAdmin bool) (*QuizReview, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}

	// Load the attempt plus job context
	var (
		appID, userID, jobID uuid.UUID
		jobTitle, jobCompany string
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
	)

	err = s.pool.QueryRow(ctx, `
		SELECT qa.application_id, qa.user_id, qa.job_id,
		       COALESCE(j.title, 'Quiz') AS job_title,
		       COALESCE(j.company, '') AS job_company,
		       qa.status, qa.score, qa.passed, qa.correct_answers,
		       qa.questions_per_quiz, qa.total_questions, qa.passing_score,
		       qa.started_at, qa.completed_at, qa.time_spent_seconds
		FROM quiz_attempts qa
		LEFT JOIN jobs j ON qa.job_id = j.id
		WHERE qa.id = $1
	`, attemptUUID).Scan(
		&appID, &userID, &jobID, &jobTitle, &jobCompany, &status,
		&score, &passed, &correctAnswers,
		&questionsPerQuiz, &totalQuestions, &passingScore,
		&startedAt, &completedAt, &timeSpent,
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
	}

	// Load every question the applicant answered for this attempt
	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.question_text, q.question_type, q.difficulty, q.options,
		       q.correct_answer, q.explanation,
		       qa.user_answer, qa.is_correct, qa.is_skipped,
		       qa.time_spent_seconds, qa.code_output, qa.created_at
		FROM quiz_answers qa
		JOIN questions q ON q.id = qa.question_id
		WHERE qa.quiz_attempt_id = $1
		ORDER BY qa.created_at ASC
	`, attemptUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answers := make([]QuizReviewAnswer, 0)
	for rows.Next() {
		var a QuizReviewAnswer
		var questionID uuid.UUID
		var options []byte
		var userAnswer, correctAnswer, explanation, codeOutput *string
		var isCorrect *bool
		var isSkipped bool
		var timeSpentSeconds int32
		var createdAt time.Time

		if err := rows.Scan(
			&questionID, &a.QuestionText, &a.QuestionType, &a.Difficulty, &options,
			&correctAnswer, &explanation,
			&userAnswer, &isCorrect, &isSkipped,
			&timeSpentSeconds, &codeOutput, &createdAt,
		); err != nil {
			return nil, err
		}

		a.QuestionID = questionID
		a.UserAnswer = userAnswer
		a.CorrectAnswer = correctAnswer
		a.IsCorrect = isCorrect
		a.IsSkipped = isSkipped
		a.Explanation = explanation
		a.TimeSpentSeconds = timeSpentSeconds
		a.CodeOutput = codeOutput
		a.CreatedAt = createdAt
		if options == nil {
			a.Options = json.RawMessage(`[]`)
		} else {
			a.Options = json.RawMessage(options)
		}

		answers = append(answers, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &QuizReview{
		ID:                attemptUUID,
		ApplicationID:     appID,
		JobApplicationID:  appID,
		UserID:            userID,
		JobID:             jobID,
		JobTitle:          jobTitle,
		JobCompany:        jobCompany,
		Title:             jobTitle,
		Status:            status,
		Score:             int4Ptr(score),
		Passed:            boolPtr(passed),
		CorrectAnswers:    int4Ptr(correctAnswers),
		TotalQuestions:    questionsPerQuiz,
		AnsweredQuestions: int32(len(answers)),
		PassingScore:      passingScore,
		StartedAt:         timestampPtr(startedAt),
		CompletedAt:       timestampPtr(completedAt),
		TimeSpentSeconds:  int4Ptr(timeSpent),
		Answers:           answers,
	}, nil
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

// GetQuizAttempt fetches the profile configuration of a specific attempt
func (s *QuizService) GetQuizAttempt(ctx context.Context, attemptID string, userID string) (*QuizAttempt, error) {
	attUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return &QuizAttempt{
		ID:     attUUID,
		UserID: userUUID,
		Title:  "Technical Core Assessment",
		Type:   "go,svelte,postgresql",
		Status: "active",
	}, nil
}

// StartQuizAttempt initializes an active session entry point in the actual database
// Updated signature to accept external IDs
func (s *QuizService) StartQuizAttempt(ctx context.Context, attemptID, userID, appID, jobID string) error {
	// 1. Check if an active attempt exists for this application
	var existingID string
	err := s.pool.QueryRow(ctx,
		"SELECT id FROM quiz_attempts WHERE application_id = $1 AND status = 'started'",
		appID).Scan(&existingID)

	if err == nil {
		// Attempt already exists! Return nil (success) or handle as "Resume"
		log.Println("Active attempt already exists, resuming:", existingID)
		return nil
	}

	// 2. Only proceed with INSERT if no attempt was found
	attemptUUID := uuid.MustParse(attemptID)
	appUUID := uuid.MustParse(appID)
	userUUID := uuid.MustParse(userID)
	jobUUID := uuid.MustParse(jobID)

	query := `
        INSERT INTO quiz_attempts (
            id, application_id, user_id, job_id, total_questions, 
            questions_per_quiz, passing_score, status, 
            started_at, last_activity_at, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, 10, 10, 70, 'started', NOW(), NOW(), NOW(), NOW())
    `

	_, err = s.pool.Exec(ctx, query, attemptUUID, appUUID, userUUID, jobUUID)
	return err
}

func (s *QuizService) GetNextQuestion(ctx context.Context, attemptID string, userID string) (map[string]interface{}, error) {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, err
	}

	// Get quiz attempt config and job tags
	var questionsPerQuiz int
	var answeredCount int
	var jobID uuid.UUID
	err = s.pool.QueryRow(ctx, `
        SELECT qp.questions_per_quiz, qp.job_id,
               (SELECT COUNT(*) FROM quiz_answers qa WHERE qa.quiz_attempt_id = qp.id)
        FROM quiz_attempts qp
        WHERE qp.id = $1
    `, attemptUUID).Scan(&questionsPerQuiz, &jobID, &answeredCount)
	if err != nil {
		log.Printf("GetNextQuestion ERROR fetching attempt %s: %v", attemptID, err)
		return nil, err
	}

	if answeredCount >= questionsPerQuiz {
		return map[string]interface{}{
			"status":  "finished",
			"message": "You have answered all questions in this quiz",
		}, nil
	}

	// Get tag names for this job (lowercase for case-insensitive matching)
	tagRows, err := s.pool.Query(ctx, `
		SELECT LOWER(t.name) FROM tags t
		JOIN job_tags jt ON jt.tag_id = t.id
		WHERE jt.job_id = $1
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job tags: %w", err)
	}
	defer tagRows.Close()

	var tagNames []string
	for tagRows.Next() {
		var name string
		if err := tagRows.Scan(&name); err != nil {
			return nil, err
		}
		tagNames = append(tagNames, name)
	}

	if len(tagNames) == 0 {
		log.Printf("Job %s has no tags, falling back to all questions", jobID)
	}

	// Select a random unanswered question matching the job's tags
	// Skip coding_challenge questions that have no data in coding_questions table
	query := `
        SELECT q.id, q.question_text, q.question_type, q.options, q.correct_answer, q.difficulty, q.time_limit_seconds
        FROM questions q
        WHERE NOT EXISTS (
            SELECT 1 FROM quiz_answers qa 
            WHERE qa.question_id = q.id AND qa.quiz_attempt_id = $1
        )
        AND (q.question_type != 'coding_challenge' OR EXISTS (
            SELECT 1 FROM coding_questions cq WHERE cq.question_id = q.id
        ))
    `

	var args []interface{}
	args = append(args, attemptUUID)

	if len(tagNames) > 0 {
		query += ` AND EXISTS (
            SELECT 1 FROM unnest(q.tags) qt WHERE LOWER(qt) = ANY($2::text[])
        )`
		args = append(args, tagNames)
	}

	query += ` ORDER BY RANDOM() LIMIT 1`

	var id uuid.UUID
	var qText, qType, difficulty string
	var options, correctAnswer *string
	var timeLimitSeconds int

	err = s.pool.QueryRow(ctx, query, args...).Scan(&id, &qText, &qType, &options, &correctAnswer, &difficulty, &timeLimitSeconds)
	if err != nil {
		log.Printf("GetNextQuestion ERROR for attemptID=%s: %v", attemptID, err)
		return nil, err
	}

	qMap := map[string]interface{}{
		"id":                 id.String(),
		"question_text":      qText,
		"question_type":      qType,
		"difficulty":         difficulty,
		"time_limit_seconds": timeLimitSeconds,
	}

	if options != nil {
		qMap["options"] = json.RawMessage(*options)
	} else {
		qMap["options"] = json.RawMessage(`[]`)
	}

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
			// Filter out hidden test cases before sending to frontend
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

			details := map[string]interface{}{
				"language":             lang,
				"code_template":        codeTemplate,
				"test_cases":           json.RawMessage(filteredTests),
				"execution_time_limit": execTimeLimit,
				"memory_limit":         memLimit,
			}
			qMap["coding_details"] = details
		}
	}

	return qMap, nil
}

// SaveQuizAnswer updates the database state by cataloging the submitted single question response
// Inside service/quiz_service.go

func (s *QuizService) SaveQuizAnswer(ctx context.Context, attemptID, userID, questionID, answer string, timeSpent int, isSkipped bool) error {
	log.Printf("DEBUG: Validating and saving answer for QuestionID: %s", questionID)

	// 1. Convert strings to UUIDs
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}
	questionUUID, err := uuid.Parse(questionID)
	if err != nil {
		return fmt.Errorf("invalid question ID: %w", err)
	}

	// 2. Fetch the correct answer and question type from the 'questions' table
	var dbCorrectAnswer *string
	var timeLimitSeconds int
	var questionType string
	err = s.pool.QueryRow(ctx, "SELECT correct_answer, time_limit_seconds, question_type::text FROM questions WHERE id = $1", questionUUID).Scan(&dbCorrectAnswer, &timeLimitSeconds, &questionType)
	if err != nil {
		return fmt.Errorf("failed to fetch correct answer for validation: %w", err)
	}

	// 3. Enforce time limit
	if timeLimitSeconds > 0 && timeSpent > timeLimitSeconds {
		return fmt.Errorf("time limit exceeded for this question (%d seconds)", timeLimitSeconds)
	}

	// 4. Determine if the user's answer is correct
	isCodingQuestion := questionType == "coding_challenge"
	isCorrect := false
	if isCodingQuestion {
		// For coding questions, preserve the existing is_correct value
		// (set by RunQuizCode based on test execution results).
		// We cannot determine correctness here since correct_answer is NULL.
		var existingCorrect *bool
		_ = s.pool.QueryRow(ctx,
			"SELECT is_correct FROM quiz_answers WHERE quiz_attempt_id = $1 AND question_id = $2",
			attemptUUID, questionUUID).Scan(&existingCorrect)
		if existingCorrect != nil {
			isCorrect = *existingCorrect
		}
	} else {
		isCorrect = dbCorrectAnswer != nil && answer == *dbCorrectAnswer
	}

	// 5. Perform the INSERT
	newAnswerID := uuid.New()

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
    `

	_, err = s.pool.Exec(ctx, query, newAnswerID, attemptUUID, questionUUID, answer, timeSpent, isSkipped, isCorrect)

	return err
}

// RunQuizCode executes user code against visible test cases for a coding_challenge question
func (s *QuizService) RunQuizCode(ctx context.Context, attemptID, questionID, language, code string) (*models.ExecuteResponse, error) {
	_, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, fmt.Errorf("invalid attempt ID: %w", err)
	}

	questionUUID, err := uuid.Parse(questionID)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %w", err)
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
		`, attemptID, questionUUID, code, isCorrect, resp.Stdout, resp.TimeMs)
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
	`, attemptID, questionUUID, code, isCorrect, resp.Stdout, resp.TimeMs)
	if upsertErr != nil {
		log.Printf("RunQuizCode: failed to save quiz_answer: %v", upsertErr)
	}

	return resp, nil
}

// SubmitQuizAttempt calculates and locks progress finality status safely
func (s *QuizService) SubmitQuizAttempt(ctx context.Context, attemptID string, userID string, tags []string) error {
	attemptUUID, err := uuid.Parse(attemptID)
	if err != nil {
		return fmt.Errorf("invalid attempt ID: %w", err)
	}

	// Check if already completed
	var currentStatus string
	err = s.pool.QueryRow(ctx, "SELECT status FROM quiz_attempts WHERE id = $1", attemptUUID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("could not find quiz attempt: %w", err)
	}
	if currentStatus == "completed" {
		return fmt.Errorf("quiz attempt %s is already completed", attemptID)
	}

	// Start a database transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get the APPLICATION_ID associated with this quiz
	var appID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT application_id FROM quiz_attempts WHERE id = $1", attemptUUID).Scan(&appID)
	if err != nil {
		return fmt.Errorf("could not find application_id for this quiz attempt: %w", err)
	}

	// 2. Calculate score
	var correctCount int
	var totalQuestions int
	err = tx.QueryRow(ctx,
		"SELECT COUNT(*) FILTER (WHERE is_correct = true), COUNT(*) FROM quiz_answers WHERE quiz_attempt_id = $1",
		attemptUUID).Scan(&correctCount, &totalQuestions)
	if err != nil {
		return err
	}

	score := 0.0
	if totalQuestions > 0 {
		score = (float64(correctCount) / float64(totalQuestions)) * 100
	}

	// 3. Fetch passing_score to determine pass/fail
	var passingScore int
	err = tx.QueryRow(ctx, "SELECT passing_score FROM quiz_attempts WHERE id = $1", attemptUUID).Scan(&passingScore)
	if err != nil {
		return fmt.Errorf("failed to fetch passing_score: %w", err)
	}

	passed := int(score) >= passingScore

	// 4. Update Quiz status to completed
	_, err = tx.Exec(ctx,
		"UPDATE quiz_attempts SET status = 'completed', score = $2, correct_answers = $3, passed = $4, completed_at = NOW(), updated_at = NOW() WHERE id = $1",
		attemptUUID, int(score), correctCount, passed)
	if err != nil {
		return err
	}

	// 5. Update the actual application status
	_, err = tx.Exec(ctx, `
        UPDATE job_applications 
        SET status = 'quiz_completed', 
            quiz_score = $1, 
            quiz_completed_at = NOW(),
            quiz_passed = $2,
            updated_at = NOW()
        WHERE id = $3`,
		score, passed, appID)
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
