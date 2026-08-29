package service

import (
	"context"
	"time"

	"github.com/Izone-hub/talent-backend/database"
)

type AdminService struct {
	queries *database.Queries
}

func NewAdminService(db database.DBTX) *AdminService {
	return &AdminService{
		queries: database.New(db),
	}
}

type DashboardStats struct {
	TotalUsers           int32 `json:"total_users"`
	ActiveJobs           int32 `json:"active_jobs"`
	PendingApplications  int32 `json:"pending_applications"`
	TotalApplications    int32 `json:"total_applications"`
	NewUsersToday        int32 `json:"new_users_today"`
	NewApplicationsToday int32 `json:"new_applications_today"`
}

type RecentActivityItem struct {
	GithubUsername string  `json:"github_username"`
	AvatarURL      *string `json:"avatar_url"`
	Status         string  `json:"status"`
	SubmittedAt    *string `json:"submitted_at"`
	JobTitle       string  `json:"job_title"`
}

func (s *AdminService) GetDashboard(ctx context.Context) (*DashboardStats, error) {
	stats, err := s.queries.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	return &DashboardStats{
		TotalUsers:           stats.TotalUsers,
		ActiveJobs:           stats.ActiveJobs,
		PendingApplications:  stats.PendingApplications,
		TotalApplications:    stats.TotalApplications,
		NewUsersToday:        stats.NewUsersToday,
		NewApplicationsToday: stats.NewApplicationsToday,
	}, nil
}

// CompanySettings represents the API response for company settings

type CompanySettingsResponse struct {
	CompanyName     string `json:"company_name"`
	CompanyLogo     string `json:"company_logo"`
	CompanyWebsite  string `json:"company_website"`
	CompanyLocation string `json:"company_location"`
}

func (s *AdminService) GetCompanySettings(ctx context.Context) (*CompanySettingsResponse, error) {
	row, err := s.queries.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}
	return &CompanySettingsResponse{
		CompanyName:     row.CompanyName,
		CompanyLogo:     row.CompanyLogo,
		CompanyWebsite:  row.CompanyWebsite,
		CompanyLocation: row.CompanyLocation,
	}, nil
}

func (s *AdminService) UpdateCompanySettings(ctx context.Context, settings CompanySettingsResponse) (*CompanySettingsResponse, error) {
	row, err := s.queries.UpdateCompanySettings(ctx, database.UpdateCompanySettingsParams{
		CompanyName:     settings.CompanyName,
		CompanyLogo:     settings.CompanyLogo,
		CompanyWebsite:  settings.CompanyWebsite,
		CompanyLocation: settings.CompanyLocation,
	})
	if err != nil {
		return nil, err
	}
	return &CompanySettingsResponse{
		CompanyName:     row.CompanyName,
		CompanyLogo:     row.CompanyLogo,
		CompanyWebsite:  row.CompanyWebsite,
		CompanyLocation: row.CompanyLocation,
	}, nil
}

func (s *AdminService) GetRecentActivity(ctx context.Context, limit int32) ([]RecentActivityItem, error) {
	rows, err := s.queries.GetRecentActivity(ctx, limit)
	if err != nil {
		return nil, err
	}
	items := make([]RecentActivityItem, 0, len(rows))
	for _, r := range rows {
		var avatarURL *string
		if r.AvatarUrl.Valid {
			avatarURL = &r.AvatarUrl.String
		}
		var submittedAt *string
		if r.SubmittedAt.Valid {
			t := r.SubmittedAt.Time.Format(time.RFC3339)
			submittedAt = &t
		}
		items = append(items, RecentActivityItem{
			GithubUsername: r.GithubUsername,
			AvatarURL:      avatarURL,
			Status:         string(r.Status),
			SubmittedAt:    submittedAt,
			JobTitle:       r.JobTitle,
		})
	}
	return items, nil
}

// CategoryJobCount represents a category with its job count for the public home page.

type CategoryJobCount struct {
	Category string `json:"category"`
	JobCount int32  `json:"job_count"`
}

func (s *AdminService) GetJobCountsByCategory(ctx context.Context) ([]CategoryJobCount, error) {
	rows, err := s.queries.GetJobCountsByCategory(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]CategoryJobCount, 0, len(rows))
	for _, r := range rows {
		items = append(items, CategoryJobCount{
			Category: string(r.Category),
			JobCount: r.JobCount,
		})
	}
	return items, nil
}
