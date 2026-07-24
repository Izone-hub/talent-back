package controller

import (
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