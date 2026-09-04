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

// RecentActivityPagination contains pagination info for recent activity.
type RecentActivityPagination struct {
	Limit   int32 `json:"limit"`
	Offset  int32 `json:"offset"`
	HasMore bool  `json:"has_more"`
	Total   int32 `json:"total"`
}

// DashboardResponse is the combined response for the admin dashboard endpoint.
// It contains both statistics and recent activity in a single response.
type DashboardResponse struct {
	Stats                   DashboardStats          `json:"stats"`
	RecentActivity          []RecentActivityItem    `json:"recent_activity"`
	RecentActivityPagination *RecentActivityPagination `json:"recent_activity_pagination,omitempty"`
}

type RecentActivityItem struct {
	GithubUsername string  `json:"github_username"`
	AvatarURL      *string `json:"avatar_url"`
	Status         string  `json:"status"`
	SubmittedAt    *string `json:"submitted_at"`
	JobTitle       string  `json:"job_title"`
}

func (s *AdminService) GetDashboard(ctx context.Context) (*DashboardResponse, error) {
	stats, err := s.queries.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch first page of recent activity (10 items)
	const defaultLimit = int32(10)
	activity, err := s.GetRecentActivity(ctx, defaultLimit)
	if err != nil {
		activity = []RecentActivityItem{}
	}

	// Get total count for pagination info
	total, err := s.queries.CountRecentActivity(ctx)
	if err != nil {
		total = 0
	}

	hasMore := total > defaultLimit

	return &DashboardResponse{
		Stats: DashboardStats{
			TotalUsers:           stats.TotalUsers,
			ActiveJobs:           stats.ActiveJobs,
			PendingApplications:  stats.PendingApplications,
			TotalApplications:    stats.TotalApplications,
			NewUsersToday:        stats.NewUsersToday,
			NewApplicationsToday: stats.NewApplicationsToday,
		},
		RecentActivity: activity,
		RecentActivityPagination: &RecentActivityPagination{
			Limit:   defaultLimit,
			Offset:  0,
			HasMore: hasMore,
			Total:   total,
		},
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
	return s.mapActivityRows(rows), nil
}

// GetRecentActivityPage returns a paginated slice of recent activity items.
func (s *AdminService) GetRecentActivityPage(ctx context.Context, limit, offset int32) ([]RecentActivityItem, error) {
	rows, err := s.queries.GetRecentActivityPage(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.mapActivityRows(rows), nil
}

// CountRecentActivity returns the total number of recent activity items.
func (s *AdminService) CountRecentActivity(ctx context.Context) (int32, error) {
	return s.queries.CountRecentActivity(ctx)
}

func (s *AdminService) mapActivityRows(rows []database.GetRecentActivityRow) []RecentActivityItem {
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
	return items
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
