package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ApplicationService struct {
	queries *database.Queries
}

func NewApplicationService(db database.DBTX) *ApplicationService {
	return &ApplicationService{
		queries: database.New(db),
	}
}

func (s *ApplicationService) ApplyForJob(ctx context.Context, jobID uuid.UUID, claims *Claims) (database.JobApplication, error) {
	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobID[:])
	pgJobID.Valid = true

	// Check if job exists and is published
	job, err := s.queries.GetJobByID(ctx, pgJobID)
	if err != nil {
		return database.JobApplication{}, fmt.Errorf("job not found: %w", err)
	}
	if job.Status != database.JobStatusPublished {
		return database.JobApplication{}, fmt.Errorf("job is not accepting applications")
	}

	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], claims.UserID[:])
	pgUserID.Valid = true

	hasApplied, err := s.queries.HasUserApplied(ctx, database.HasUserAppliedParams{
		JobID:  pgJobID,
		UserID: pgUserID,
	})
	if err != nil {
		return database.JobApplication{}, fmt.Errorf("failed to check existing application: %w", err)
	}
	if hasApplied {
		return database.JobApplication{}, fmt.Errorf("user has already applied for this job")
	}

	user, err := s.queries.GetUserByID(ctx, pgUserID)
	if err != nil {
		return database.JobApplication{}, fmt.Errorf("failed to retrieve user details: %w", err)
	}

	now := time.Now()
	var pgNow pgtype.Timestamp
	pgNow.Time = now
	pgNow.Valid = true

	app, err := s.queries.CreateApplication(ctx, database.CreateApplicationParams{
		JobID:              pgJobID,
		UserID:             pgUserID,
		GithubUsername:     claims.GithubUsername,
		GithubID:           claims.GithubID,
		ApplicantEmail:     user.Email,
		ApplicantName:      user.Name,
		ApplicantAvatarUrl: user.AvatarUrl,
		Status:             database.ApplicationStatusSubmitted,
		SubmittedAt:        pgNow,
	})
	if err != nil {
		return database.JobApplication{}, fmt.Errorf("failed to create application: %w", err)
	}

	quizAttempt, err := s.queries.CreateQuizAttempt(ctx, database.CreateQuizAttemptParams{
		ApplicationID:    app.ID,
		UserID:           pgUserID,
		JobID:            pgJobID,
		TotalQuestions:   10,
		QuestionsPerQuiz: 10,
		TimeLimitMinutes: pgtype.Int4{Int32: 30, Valid: true},
		PassingScore:     70,
	})
	if err != nil {
		fmt.Printf("ERROR CreateQuizAttempt: %v\n", err)
		return app, nil
	}

	quizUUID, err := uuid.FromBytes(quizAttempt.ID.Bytes[:])
	if err != nil {
		fmt.Printf("ERROR converting quiz ID: %v\n", err)
	} else {
		updatedApp, err := s.queries.ApplicationStartQuiz(ctx, database.ApplicationStartQuizParams{
			ID:     app.ID,
			QuizID: quizUUID,
		})
		if err != nil {
			fmt.Printf("ERROR ApplicationStartQuiz: %v\n", err)
		} else {
			return updatedApp, nil
		}
	}

	return app, nil
}

func (s *ApplicationService) GetMyApplications(ctx context.Context, userID uuid.UUID) ([]database.ListApplicationsByUserRow, error) {
	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], userID[:])
	pgUserID.Valid = true

	return s.queries.ListApplicationsByUser(ctx, database.ListApplicationsByUserParams{
		UserID: pgUserID,
		Limit:  100,
		Offset: 0,
	})
}

func (s *ApplicationService) GetApplicationsForJob(ctx context.Context, jobID uuid.UUID, limit, offset int32) ([]database.ListApplicationsByJobRow, error) {
	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobID[:])
	pgJobID.Valid = true

	return s.queries.ListApplicationsByJob(ctx, database.ListApplicationsByJobParams{
		JobID:  pgJobID,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *ApplicationService) GetApplicationDetail(ctx context.Context, appID uuid.UUID) (database.GetApplicationWithDetailsRow, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.GetApplicationWithDetails(ctx, pgAppID)
}

func (s *ApplicationService) StartReview(ctx context.Context, appID uuid.UUID, reviewerID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.StartReview(ctx, database.StartReviewParams{
		ID:         pgAppID,
		ReviewedBy: reviewerID,
	})
}

func (s *ApplicationService) ShortlistApplication(ctx context.Context, appID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.ShortlistApplication(ctx, pgAppID)
}

func (s *ApplicationService) MarkInterviewed(ctx context.Context, appID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.MarkInterviewed(ctx, pgAppID)
}

func (s *ApplicationService) AcceptApplication(ctx context.Context, appID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.AcceptApplication(ctx, pgAppID)
}

func (s *ApplicationService) RejectApplication(ctx context.Context, appID uuid.UUID, reason, feedback string) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.RejectApplication(ctx, database.RejectApplicationParams{
		ID:               pgAppID,
		RejectionReason:  pgtype.Text{String: reason, Valid: reason != ""},
		EmployerFeedback: pgtype.Text{String: feedback, Valid: feedback != ""},
	})
}

func (s *ApplicationService) WithdrawApplication(ctx context.Context, appID uuid.UUID, userID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], userID[:])
	pgUserID.Valid = true

	return s.queries.WithdrawApplication(ctx, database.WithdrawApplicationParams{
		ID:     pgAppID,
		UserID: pgUserID,
	})
}

func (s *ApplicationService) ListApplicationsByStatus(ctx context.Context, status database.ApplicationStatus, limit, offset int32) ([]database.JobApplication, error) {
	return s.queries.ListApplicationsByStatus(ctx, database.ListApplicationsByStatusParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *ApplicationService) GetApplicationCountsByJob(ctx context.Context, jobID uuid.UUID) (database.GetApplicationCountsByJobRow, error) {
	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobID[:])
	pgJobID.Valid = true

	return s.queries.GetApplicationCountsByJob(ctx, pgJobID)
}

func (s *ApplicationService) AddEmployerFeedback(ctx context.Context, appID uuid.UUID, feedback string) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.AddEmployerFeedback(ctx, database.AddEmployerFeedbackParams{
		ID:               pgAppID,
		EmployerFeedback: pgtype.Text{String: feedback, Valid: feedback != ""},
	})
}

func (s *ApplicationService) GetRecentApplications(ctx context.Context, limit int32) ([]database.JobApplication, error) {
	return s.queries.GetRecentApplications(ctx, limit)
}

type ApplicationInformation struct {
	Application models.JobApplication `json:"application"`
	AISummary   models.AiSummary      `json:"ai_summary,omitempty"`
	QuizAttempt models.QuizAttempt    `json:"quiz_attempt,omitempty"`
}

func dbAppToModel(app database.JobApplication) models.JobApplication {
	model := models.JobApplication{}
	copy(model.ID[:], app.ID.Bytes[:])
	copy(model.JobID[:], app.JobID.Bytes[:])
	copy(model.UserID[:], app.UserID.Bytes[:])

	model.GithubUsername = app.GithubUsername
	model.GithubID = app.GithubID
	model.ApplicantEmail = pgTextToStrPtr(app.ApplicantEmail)
	model.ApplicantName = pgTextToStrPtr(app.ApplicantName)
	model.ApplicantAvatarURL = pgTextToStrPtr(app.ApplicantAvatarUrl)
	model.CoverLetter = pgTextToStrPtr(app.CoverLetter)
	if app.ProposedSalary.Valid {
		v := int(app.ProposedSalary.Int32)
		model.ProposedSalary = &v
	}
	model.ProposedSalaryCurrency = pgTextToString(app.ProposedSalaryCurrency)
	model.AvailabilityDate = pgTimestampToTimePtr(app.AvailabilityDate)
	model.PortfolioURL = pgTextToStrPtr(app.PortfolioUrl)
	model.LinkedinURL = pgTextToStrPtr(app.LinkedinUrl)
	model.Notes = pgTextToStrPtr(app.Notes)
	model.Status = models.ApplicationStatus(app.Status)
	model.SubmittedAt = pgTimestampToTimePtr(app.SubmittedAt)
	model.ReviewedAt = pgTimestampToTimePtr(app.ReviewedAt)

	if app.ReviewedBy != uuid.Nil {
		v := app.ReviewedBy
		model.ReviewedBy = &v
	}

	model.EmployerFeedback = pgTextToStrPtr(app.EmployerFeedback)
	model.RejectionReason = pgTextToStrPtr(app.RejectionReason)

	if app.QuizID != uuid.Nil {
		v := app.QuizID
		model.QuizID = &v
	}
	if app.QuizScore.Valid {
		v := int(app.QuizScore.Int32)
		model.QuizScore = &v
	}
	model.QuizCompletedAt = pgTimestampToTimePtr(app.QuizCompletedAt)
	if app.QuizPassed.Valid {
		model.QuizPassed = &app.QuizPassed.Bool
	}
	model.CanViewAISummary = app.CanViewAiSummary.Bool && app.CanViewAiSummary.Valid
	model.CreatedAt = pgTimestampToTime(app.CreatedAt)
	model.UpdatedAt = pgTimestampToTime(app.UpdatedAt)

	return model
}

func dbAiSummaryToModel(s database.AiSummary) models.AiSummary {
	model := models.AiSummary{}
	copy(model.ID[:], s.ID.Bytes[:])
	copy(model.UserID[:], s.UserID[:])

	if len(s.Summary) > 0 {
		model.Summary = json.RawMessage(s.Summary)
	}
	model.Strengths = pgTextToStrPtr(s.Strengths)
	model.Weaknesses = pgTextToStrPtr(s.Weaknesses)
	model.Model = pgTextToStrPtr(s.Model)
	model.CreatedAt = pgTimestampToTimePtr(s.CreatedAt)
	if s.CvVersion.Valid {
		v := int(s.CvVersion.Int32)
		model.CvVersion = &v
	}
	return model
}

func dbQuizAttemptToModel(q database.QuizAttempt) models.QuizAttempt {
	model := models.QuizAttempt{}
	copy(model.ID[:], q.ID.Bytes[:])
	copy(model.ApplicationID[:], q.ApplicationID.Bytes[:])
	copy(model.UserID[:], q.UserID.Bytes[:])
	copy(model.JobID[:], q.JobID.Bytes[:])

	model.TotalQuestions = int(q.TotalQuestions)
	model.QuestionsPerQuiz = int(q.QuestionsPerQuiz)
	if q.TimeLimitMinutes.Valid {
		v := int(q.TimeLimitMinutes.Int32)
		model.TimeLimitMinutes = &v
	}
	model.PassingScore = int(q.PassingScore)
	model.Status = models.QuizAttemptStatus(q.Status)
	model.StartedAt = pgTimestampToTime(q.StartedAt)
	model.CompletedAt = pgTimestampToTimePtr(q.CompletedAt)
	model.LastActivityAt = pgTimestampToTime(q.LastActivityAt)
	if q.Score.Valid {
		v := int(q.Score.Int32)
		model.Score = &v
	}
	model.CorrectAnswers = int(q.CorrectAnswers.Int32)
	model.IncorrectAnswers = int(q.IncorrectAnswers.Int32)
	model.SkippedQuestions = int(q.SkippedQuestions.Int32)
	if q.Passed.Valid {
		model.Passed = &q.Passed.Bool
	}
	model.TimeSpentSeconds = int(q.TimeSpentSeconds.Int32)
	model.AutoSaveIntervalSeconds = int(q.AutoSaveIntervalSeconds.Int32)
	model.CreatedAt = pgTimestampToTime(q.CreatedAt)
	model.UpdatedAt = pgTimestampToTime(q.UpdatedAt)
	return model
}

func (s *ApplicationService) GetApplicationInformation(ctx context.Context, appID uuid.UUID) (ApplicationInformation, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	app, err := s.queries.GetApplicationByID(ctx, pgAppID)
	if err != nil {
		return ApplicationInformation{}, fmt.Errorf("application not found: %w", err)
	}

	userID, err := uuid.FromBytes(app.UserID.Bytes[:])
	if err != nil {
		return ApplicationInformation{}, fmt.Errorf("invalid user ID in application: %w", err)
	}

	aiSummary, err := s.queries.GetLatestAISummary(ctx, userID)
	if err != nil {
		aiSummary = database.AiSummary{}
	}

	quizAttempt, err := s.queries.GetQuizAttemptByApplication(ctx, pgAppID)
	if err != nil {
		quizAttempt = database.QuizAttempt{}
	}

	return ApplicationInformation{
		Application: dbAppToModel(app),
		AISummary:   dbAiSummaryToModel(aiSummary),
		QuizAttempt: dbQuizAttemptToModel(quizAttempt),
	}, nil
}
