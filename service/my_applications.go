package service

import (
	"context"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ApplicationListItem is the UI-ready shape of one row on the "My Applications"
// page. The field names intentionally mirror the previous payload so existing
// consumers keep working, but only the fields the UI actually renders are
// included (job snapshot columns come from the single SQL join).
type ApplicationListItem struct {
	ID               string
	JobID            string
	JobTitle         string
	JobCompany       string
	JobStatus        string
	JobLocation      *string
	JobType          string
	Status           string
	SubmittedAt      *string
	UpdatedAt        string
	QuizID           *string
	QuizScore        *int
	QuizPassed       *bool
	EmployerFeedback *string
	ApplicantEmail   *string
}

// MyApplicationsStats are aggregated in SQL over exactly the returned page.
type MyApplicationsStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// MyApplicationsResponse is the structured payload of GET /applications/my.
type MyApplicationsResponse struct {
	Applications []ApplicationListItem `json:"applications"`
	Stats        MyApplicationsStats   `json:"stats"`
}

// myApplicationsPageSize bounds the number of applications returned per
// request (kept aligned with the previous limit of 100).
const myApplicationsPageSize = 100

// GetMyApplications returns the applications shown on the user's dashboard
// together with precomputed stats. All retrieval, joining, accepted-job
// filtering (from users.acceptance_job_id — the single source of truth) and
// stats aggregation happen in the SQL query; this method only shapes rows.
func (s *ApplicationService) GetMyApplications(ctx context.Context, userID uuid.UUID) (MyApplicationsResponse, error) {
	rows, err := s.queries.ListMyApplicationsDashboard(ctx, database.ListMyApplicationsDashboardParams{
		UserID: uuidToPgUUID(userID),
		Limit:  myApplicationsPageSize,
		Offset: 0,
	})
	if err != nil {
		return MyApplicationsResponse{}, err
	}

	res := MyApplicationsResponse{
		Applications: make([]ApplicationListItem, 0, len(rows)),
	}
	for _, r := range rows {
		res.Applications = append(res.Applications, applicationListItemFromRow(r))
	}
	if len(rows) > 0 {
		first := rows[0]
		res.Stats = MyApplicationsStats{
			Total:    int(first.Total),
			Active:   int(first.Active),
			Accepted: int(first.Accepted),
			Rejected: int(first.Rejected),
		}
	}
	return res, nil
}

// GetUserApplications returns every application a user has submitted across
// all jobs (admin view). Unlike the user's own dashboard query it includes the
// full application history with no acceptance filtering.
func (s *ApplicationService) GetUserApplications(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]ApplicationListItem, error) {
	rows, err := s.queries.ListApplicationsByUser(ctx, database.ListApplicationsByUserParams{
		UserID: uuidToPgUUID(userID),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]ApplicationListItem, 0, len(rows))
	for _, r := range rows {
		item := ApplicationListItem{
			ID:               uuidToString(r.ID),
			JobID:            uuidToString(r.JobID),
			JobTitle:         r.JobTitle,
			JobCompany:       r.JobCompany,
			JobStatus:        string(r.JobStatus),
			JobLocation:      pgTextToStrPtr(r.JobLocation),
			JobType:          string(r.JobType),
			Status:           string(r.Status),
			UpdatedAt:        timestampToString(r.UpdatedAt),
			QuizID:           uuidOrNil(r.QuizID),
			QuizScore:        pgInt4ToIntPtr(r.QuizScore),
			QuizPassed:       boolOrNil(r.QuizPassed),
			EmployerFeedback: pgTextToStrPtr(r.EmployerFeedback),
			ApplicantEmail:   pgTextToStrPtr(r.ApplicantEmail),
		}
		if r.SubmittedAt.Valid {
			s := timestampToString(r.SubmittedAt)
			item.SubmittedAt = &s
		}
		items = append(items, item)
	}
	return items, nil
}

func applicationListItemFromRow(r database.ListMyApplicationsDashboardRow) ApplicationListItem {
	item := ApplicationListItem{
		ID:               uuidToString(r.ID),
		JobID:            uuidToString(r.JobID),
		JobTitle:         r.JobTitle,
		JobCompany:       r.JobCompany,
		JobStatus:        string(r.JobStatus),
		JobLocation:      pgTextToStrPtr(r.JobLocation),
		JobType:          string(r.JobType),
		Status:           string(r.Status),
		UpdatedAt:        timestampToString(r.UpdatedAt),
		QuizID:           uuidOrNil(r.QuizID),
		QuizScore:        pgInt4ToIntPtr(r.QuizScore),
		QuizPassed:       boolOrNil(r.QuizPassed),
		EmployerFeedback: pgTextToStrPtr(r.EmployerFeedback),
		ApplicantEmail:   pgTextToStrPtr(r.ApplicantEmail),
	}
	if r.SubmittedAt.Valid {
		s := timestampToString(r.SubmittedAt)
		item.SubmittedAt = &s
	}
	return item
}

// uuidToString renders a NOT NULL pgtype.UUID as a plain string.
func uuidToString(id pgtype.UUID) string {
	return pgUUIDToUUID(id).String()
}

// uuidOrNil renders a nullable uuid.UUID as a *string, nil when unset.
func uuidOrNil(id uuid.UUID) *string {
	if id == uuid.Nil {
		return nil
	}
	s := id.String()
	return &s
}

// boolOrNil renders a nullable pgtype.Bool as a *bool, nil when unset.
func boolOrNil(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}

// timestampToString renders a pgtype.Timestamp as an RFC3339 string in UTC.
func timestampToString(t pgtype.Timestamp) string {
	return t.Time.UTC().Format(time.RFC3339)
}
