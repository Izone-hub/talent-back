package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/Izone-hub/talent-backend/database"
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
	// 1. Check if job exists
	var pgJobID pgtype.UUID
	copy(pgJobID.Bytes[:], jobID[:])
	pgJobID.Valid = true

	var pgUserID pgtype.UUID
	copy(pgUserID.Bytes[:], claims.UserID[:])
	pgUserID.Valid = true

	// Check if already applied
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

	// Retrieve user profile to populate applicant details
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

	// 3. Create Quiz attempt and link it back to the application
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
		// Application was created but quiz failed — still return the application
		fmt.Printf("ERROR CreateQuizAttempt: %v\n", err)
		return app, nil
	}
 
	// Link the quiz_id back to the application and set status to quiz_started
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

// GetMyApplications fetches all applications for a candidate with full details
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

// GetApplicationsForJob fetches all applications for a specific job (Employer view)
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

// AcceptApplication marks an application as accepted (Employer action)
func (s *ApplicationService) AcceptApplication(ctx context.Context, appID uuid.UUID) (database.JobApplication, error) {
	var pgAppID pgtype.UUID
	copy(pgAppID.Bytes[:], appID[:])
	pgAppID.Valid = true

	return s.queries.AcceptApplication(ctx, pgAppID)
}
