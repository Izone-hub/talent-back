package service

import (
	"context"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
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
	Stats                    DashboardStats            `json:"stats"`
	RecentActivity           []RecentActivityItem      `json:"recent_activity"`
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

// AdminUser represents a user row in the admin applicants/users listing.
// It carries the same fields as models.User but omits sensitive data
// (GitHub access token / token expiry) from the JSON response.
type AdminUser struct {
	ID                   uuid.UUID  `json:"id"`
	GithubID             int64      `json:"github_id"`
	GithubUsername       string     `json:"github_username"`
	Email                *string    `json:"email,omitempty"`
	AvatarURL            *string    `json:"avatar_url,omitempty"`
	Name                 *string    `json:"name,omitempty"`
	Role                 string     `json:"role"`
	LastLoginAt          *string    `json:"last_login_at,omitempty"`
	CreatedAt            *string    `json:"created_at,omitempty"`
	UpdatedAt            *string    `json:"updated_at,omitempty"`
	PublicRepos          int        `json:"public_repos"`
	PublicGists          int        `json:"public_gists"`
	Followers            int        `json:"followers"`
	Following            int        `json:"following"`
	Hireable             bool       `json:"hireable"`
	Blog                 *string    `json:"blog,omitempty"`
	Company              *string    `json:"company,omitempty"`
	Location             *string    `json:"location,omitempty"`
	Bio                  *string    `json:"bio,omitempty"`
	TwitterUsername      *string    `json:"twitter_username,omitempty"`
	TopLanguages         []string   `json:"top_languages"`
	ContributionCount    int        `json:"contribution_count"`
	AcceptanceJobID      *uuid.UUID `json:"acceptance_job_id,omitempty"`
	Categories           []string   `json:"categories"`
}

// ListAllUsers returns a paginated list of registered users, newest first,
// together with the total count for pagination. When category is non-empty it
// lists only accepted users in that category; otherwise every user. The
// returned rows never include the GitHub access token.
func (s *AdminService) ListAllUsers(ctx context.Context, category string, limit, offset int32) ([]AdminUser, int64, error) {
	var rows []database.User
	var total int64
	var err error

	if category != "" {
		rows, err = s.queries.ListUsersByCategory(ctx, database.ListUsersByCategoryParams{
			Column1: category,
			Limit:   limit,
			Offset:  offset,
		})
		if err != nil {
			return nil, 0, err
		}
		total, err = s.queries.CountUsersByCategory(ctx, category)
	} else {
		rows, err = s.queries.ListUsers(ctx, database.ListUsersParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, 0, err
		}
		total, err = s.queries.CountUsers(ctx)
	}
	if err != nil {
		return nil, 0, err
	}

	users := make([]AdminUser, 0, len(rows))
	for _, r := range rows {
		users = append(users, adminUserFromDB(r))
	}

	return users, total, nil
}

// adminUserFromDB converts a database.User into the safe AdminUser DTO,
// omitting the GitHub access token and token expiry.
func adminUserFromDB(r database.User) AdminUser {
	categories := r.Categories
	if categories == nil {
		categories = []string{}
	}
	return AdminUser{
		ID:                pgUUIDToUUID(r.ID),
		GithubID:          r.GithubID,
		GithubUsername:    r.GithubUsername,
		Email:             pgTextToStrPtr(r.Email),
		AvatarURL:         pgTextToStrPtr(r.AvatarUrl),
		Name:              pgTextToStrPtr(r.Name),
		Role:              r.Role,
		LastLoginAt:       pgTimestampToTimePtrStr(r.LastLoginAt),
		CreatedAt:         pgTimestampToTimePtrStr(r.CreatedAt),
		UpdatedAt:         pgTimestampToTimePtrStr(r.UpdatedAt),
		PublicRepos:       int(r.PublicRepos.Int32),
		PublicGists:       int(r.PublicGists.Int32),
		Followers:         int(r.Followers.Int32),
		Following:         int(r.Following.Int32),
		Hireable:          r.Hireable.Bool,
		Blog:              pgTextToStrPtr(r.Blog),
		Company:           pgTextToStrPtr(r.Company),
		Location:          pgTextToStrPtr(r.Location),
		Bio:               pgTextToStrPtr(r.Bio),
		TwitterUsername:   pgTextToStrPtr(r.TwitterUsername),
		TopLanguages:      r.TopLanguages,
		ContributionCount: int(r.ContributionCount.Int32),
		AcceptanceJobID:   pgUUIDToUUIDPtr(r.AcceptanceJobID),
		Categories:        categories,
	}
}

// GetUser returns a single registered user for the admin user detail page,
// with sensitive fields (GitHub access token / token expiry) stripped.
func (s *AdminService) GetUser(ctx context.Context, userID uuid.UUID) (*AdminUser, error) {
	row, err := s.queries.GetUserByID(ctx, uuidToPgUUID(userID))
	if err != nil {
		return nil, err
	}
	user := adminUserFromDB(row)
	return &user, nil
}

// UserCategoryCount is one entry of the admin Users category cards: a category
// key with the number of accepted users in it.
type UserCategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// GetUserCategoryCounts returns per-category counts of accepted users for the
// admin Users page cards. Users with several categories are counted once per
// category.
func (s *AdminService) GetUserCategoryCounts(ctx context.Context) ([]UserCategoryCount, error) {
	rows, err := s.queries.CountAcceptedUsersByCategory(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]UserCategoryCount, 0, len(rows))
	for _, r := range rows {
		items = append(items, UserCategoryCount{
			Category: r.Category,
			Count:    r.Count,
		})
	}
	return items, nil
}

// ReindexUserCategories derives and stores categories for every accepted user
// whose stored categories are still empty — a one-time backfill for users
// accepted before the categories column existed. Idempotent: rows with an
// existing value (e.g. a future admin override) are left untouched. Returns
// the number of users updated.
func (s *AdminService) ReindexUserCategories(ctx context.Context) (int, error) {
	rows, err := s.queries.ListAcceptedUsers(ctx)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, u := range rows {
		if len(u.Categories) > 0 {
			continue
		}
		cats := deriveCategories(u.TopLanguages)
		if len(cats) == 0 {
			continue
		}
		if err := s.queries.SetUserCategories(ctx, database.SetUserCategoriesParams{
			ID:         u.ID,
			Categories: cats,
		}); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
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
	rows, err := s.queries.GetRecentActivityPage(ctx, database.GetRecentActivityPageParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	// The page row type mirrors GetRecentActivityRow field-for-field; convert
	// so the shared mapping below stays the single source of presentation.
	converted := make([]database.GetRecentActivityRow, 0, len(rows))
	for _, r := range rows {
		converted = append(converted, database.GetRecentActivityRow{
			GithubUsername: r.GithubUsername,
			AvatarUrl:      r.AvatarUrl,
			Status:         r.Status,
			SubmittedAt:    r.SubmittedAt,
			JobTitle:       r.JobTitle,
		})
	}
	return s.mapActivityRows(converted), nil
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

// ---------------------------------------------------------------------------
// Admin applications overview (single aggregate request, no N+1 per job)
// ---------------------------------------------------------------------------

type AdminJobApplicationCounts struct {
	TotalApplications int64 `json:"total_applications"`
	Submitted         int64 `json:"submitted"`
	QuizStarted       int64 `json:"quiz_started"`
	QuizCompleted     int64 `json:"quiz_completed"`
	UnderReview       int64 `json:"under_review"`
	Shortlisted       int64 `json:"shortlisted"`
	Interviewed       int64 `json:"interviewed"`
	Rejected          int64 `json:"rejected"`
	Withdrawn         int64 `json:"withdrawn"`
	Accepted          int64 `json:"accepted"`
}

// OverviewJob carries only the job fields the admin applications cards render,
// with the application counters already attached.
type OverviewJob struct {
	ID             uuid.UUID                 `json:"id"`
	Title          string                    `json:"title"`
	Company        string                    `json:"company"`
	Category       string                    `json:"category"`
	Location       *string                   `json:"location,omitempty"`
	JobType        string                    `json:"job_type"`
	RemotePossible bool                      `json:"remote_possible"`
	Stats          AdminJobApplicationCounts `json:"stats"`
}

type QuizCompletedCandidate struct {
	ApplicationID  uuid.UUID `json:"application_id"`
	JobID          uuid.UUID `json:"job_id"`
	QuizScore      *int      `json:"quiz_score"`
	ApplicantName  string    `json:"applicant_name"`
	ApplicantEmail string    `json:"applicant_email"`
	GithubUsername string    `json:"applicant_github_username"`
	AvatarURL      *string   `json:"applicant_avatar_url"`
	JobTitle       string    `json:"job_title"`
}

type AdminApplicationsOverview struct {
	Jobs           []OverviewJob            `json:"jobs"`
	QuizCandidates []QuizCompletedCandidate `json:"quiz_candidates"`
	Summary        AdminApplicationsSummary `json:"summary"`
}

type AdminApplicationsSummary struct {
	TotalApplicants int64 `json:"total_applicants"`
	QuizDone        int64 `json:"quiz_done"`
	Shortlisted     int64 `json:"shortlisted"`
	Accepted        int64 `json:"accepted"`
}

// GetApplicationsOverview aggregates everything the admin applications page
// needs into a single payload: every published job with its per-job counters,
// the overall summary stats, and the quiz-completed candidates list - all
// computed in SQL so the client renders without extra requests or math.
func (s *AdminService) GetApplicationsOverview(ctx context.Context, quizCandidateLimit int32) (*AdminApplicationsOverview, error) {
	rows, err := s.queries.ListPublishedJobApplicationStats(ctx)
	if err != nil {
		return nil, err
	}

	if quizCandidateLimit <= 0 {
		quizCandidateLimit = 100
	}
	candidates, err := s.queries.ListQuizCompletedCandidates(ctx, quizCandidateLimit)
	if err != nil {
		return nil, err
	}

	jobs := make([]OverviewJob, 0, len(rows))
	summary := AdminApplicationsSummary{}
	for _, r := range rows {
		job := OverviewJob{
			ID:             pgUUIDToUUID(r.JobID),
			Title:          r.Title,
			Company:        r.Company,
			Category:       string(r.Category),
			Location:       pgTextToStrPtr(r.Location),
			JobType:        string(r.JobType),
			RemotePossible: r.RemotePossible.Bool && r.RemotePossible.Valid,
			Stats: AdminJobApplicationCounts{
				TotalApplications: r.TotalApplications,
				Submitted:         r.Submitted,
				QuizStarted:       r.QuizStarted,
				QuizCompleted:     r.QuizCompleted,
				UnderReview:       r.UnderReview,
				Shortlisted:       r.Shortlisted,
				Interviewed:       r.Interviewed,
				Rejected:          r.Rejected,
				Withdrawn:         r.Withdrawn,
				Accepted:          r.Accepted,
			},
		}
		jobs = append(jobs, job)
		summary.TotalApplicants += r.TotalApplications
		summary.QuizDone += r.QuizCompleted
		summary.Shortlisted += r.Shortlisted
		summary.Accepted += r.Accepted
	}

	candidateModels := make([]QuizCompletedCandidate, 0, len(candidates))
	for _, c := range candidates {
		var score *int
		if c.QuizScore.Valid {
			s := int(c.QuizScore.Int32)
			score = &s
		}
		candidateModels = append(candidateModels, QuizCompletedCandidate{
			ApplicationID:  pgUUIDToUUID(c.ApplicationID),
			JobID:          pgUUIDToUUID(c.JobID),
			QuizScore:      score,
			ApplicantName:  c.ApplicantName,
			ApplicantEmail: c.ApplicantEmail,
			GithubUsername: c.ApplicantGithubUsername,
			AvatarURL:      emptyStrToNil(c.ApplicantAvatarUrl),
			JobTitle:       c.JobTitle,
		})
	}

	return &AdminApplicationsOverview{
		Jobs:           jobs,
		QuizCandidates: candidateModels,
		Summary:        summary,
	}, nil
}

// emptyStrToNil converts a plain string from sqlc (COALESCE(...)::text yields
// a plain string) into a *string for JSON output; empty becomes nil.
func emptyStrToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
