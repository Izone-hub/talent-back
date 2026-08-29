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
	stats, err := c.adminService.GetDashboard(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch dashboard stats: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
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

func (c *AdminController) GetRecentActivity(w http.ResponseWriter, r *http.Request) {
	limit := int32(10)
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 50 {
			limit = int32(v)
		}
	}

	activity, err := c.adminService.GetRecentActivity(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch recent activity: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, activity)
}

func (c *AdminController) GetJobCountsByCategory(w http.ResponseWriter, r *http.Request) {
	counts, err := c.adminService.GetJobCountsByCategory(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch job counts: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, counts)
}
