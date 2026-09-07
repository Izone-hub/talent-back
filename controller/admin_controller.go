package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type AdminController struct {
	adminService *service.AdminService
}

func NewAdminController(adminService *service.AdminService) *AdminController {
	return &AdminController{
		adminService: adminService,
	}
}

func (c *AdminController) GetDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.adminService.GetDashboard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch dashboard: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (c *AdminController) GetCompanySettings(w http.ResponseWriter, r *http.Request) {
	settings, err := c.adminService.GetCompanySettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch company settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (c *AdminController) UpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
	var req service.CompanySettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	settings, err := c.adminService.UpdateCompanySettings(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update company settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (c *AdminController) GetApplicationsOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := c.adminService.GetApplicationsOverview(r.Context(), 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch applications overview: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (c *AdminController) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := int32(50)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = int32(v)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	category := r.URL.Query().Get("category")

	users, total, err := c.adminService.ListAllUsers(r.Context(), category, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch users: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": users,
		"pagination": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"total":    total,
			"has_more": int64(offset+limit) < total,
		},
	})
}

// GetUserCategoryCounts returns per-category counts of accepted users for the
// admin Users page cards (admin only).
func (c *AdminController) GetUserCategoryCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := c.adminService.GetUserCategoryCounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch category counts: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"categories": counts,
	})
}

// ReindexUserCategories backfills categories for accepted users accepted
// before the categories column existed (admin only). Idempotent.
func (c *AdminController) ReindexUserCategories(w http.ResponseWriter, r *http.Request) {
	updated, err := c.adminService.ReindexUserCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reindex user categories: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User categories reindexed",
		"updated": updated,
	})
}

func (c *AdminController) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := c.adminService.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (c *AdminController) GetJobCountsByCategory(w http.ResponseWriter, r *http.Request) {
	counts, err := c.adminService.GetJobCountsByCategory(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch job counts: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, counts)
}

func (c *AdminController) GetRecentActivityPage(w http.ResponseWriter, r *http.Request) {
	limit := int32(10)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = int32(v)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	activity, err := c.adminService.GetRecentActivityPage(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch recent activity: "+err.Error())
		return
	}

	total, err := c.adminService.CountRecentActivity(r.Context())
	if err != nil {
		total = 0
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": activity,
		"pagination": map[string]interface{}{
			"limit":    limit,
			"offset":   offset,
			"has_more": offset+limit < total,
			"total":    total,
		},
	})
}
