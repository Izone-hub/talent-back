package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Izone-hub/talent-backend/service"
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
