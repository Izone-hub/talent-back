package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

type AdminController struct {
	adminService  *service.AdminService
	analyzerURL   string
	internalToken string
}

func NewAdminController(adminService *service.AdminService, analyzerURL, internalToken string) *AdminController {
	return &AdminController{
		adminService:  adminService,
		analyzerURL:   strings.TrimSuffix(analyzerURL, "/"),
		internalToken: internalToken,
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

func (c *AdminController) ListContactRequests(w http.ResponseWriter, r *http.Request) {
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

	requests, err := c.adminService.ListContactRequests(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch contact requests")
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (c *AdminController) GetContactRequest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}

	request, err := c.adminService.GetContactRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (c *AdminController) ListContactRequestsByEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	if email == "" {
		writeError(w, http.StatusBadRequest, "Email is required")
		return
	}
	requests, err := c.adminService.ListContactRequestsByEmail(r.Context(), email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch contact requests")
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (c *AdminController) ListContactRequestMessages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}
	request, err := c.adminService.GetContactRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}
	messages, err := c.adminService.ListContactRequestsByEmail(r.Context(), request.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch contact messages")
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (c *AdminController) DeleteContactRequest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}
	if err := c.adminService.DeleteContactRequest(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteContactRequestMessage deletes a single message (contact request) within
// a conversation. It validates that the target message belongs to the same
// email as the parent conversation before deleting.
func (c *AdminController) DeleteContactRequestMessage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}
	messageID, err := uuid.Parse(r.PathValue("messageId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	// Fetch the parent conversation to get the email.
	parent, err := c.adminService.GetContactRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}

	// Fetch the target message and verify it belongs to the same email.
	message, err := c.adminService.GetContactRequest(r.Context(), messageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Message not found")
		return
	}
	if !strings.EqualFold(message.Email, parent.Email) {
		writeError(w, http.StatusForbidden, "Message does not belong to this conversation")
		return
	}

	if err := c.adminService.DeleteContactRequest(r.Context(), messageID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete message")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *AdminController) UpdateContactRequestStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Status != "new" && req.Status != "read" && req.Status != "replied" && req.Status != "archived" {
		writeError(w, http.StatusBadRequest, "Invalid contact request status")
		return
	}

	request, err := c.adminService.UpdateContactRequestStatus(r.Context(), id, req.Status)
	if err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (c *AdminController) ReplyToContactRequest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid contact request ID")
		return
	}

	var req struct {
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)
	if req.Subject == "" || req.Message == "" {
		writeError(w, http.StatusBadRequest, "Subject and message are required")
		return
	}
	if len(req.Subject) > 255 || len(req.Message) > 10000 {
		writeError(w, http.StatusBadRequest, "Subject or message is too long")
		return
	}

	contact, err := c.adminService.GetContactRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Contact request not found")
		return
	}

	payload, err := json.Marshal(map[string]string{
		"type":            "contact_reply",
		"recipient_email": contact.Email,
		"recipient_name":  strings.TrimSpace(contact.FirstName + " " + contact.LastName),
		"subject":         req.Subject,
		"message":         req.Message,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to prepare reply")
		return
	}

	analyzerRequest, err := http.NewRequestWithContext(
		r.Context(), http.MethodPost,
		c.analyzerURL+"/api/v1/notifications/contact-reply",
		bytes.NewReader(payload),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to prepare reply")
		return
	}
	analyzerRequest.Header.Set("Content-Type", "application/json")
	if c.internalToken != "" {
		analyzerRequest.Header.Set("X-Internal-Service-Token", c.internalToken)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(analyzerRequest)
	if err != nil {
		log.Printf("ERROR: failed to send contact reply for %s: %v", id, err)
		writeError(w, http.StatusBadGateway, "Reply service is temporarily unavailable")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		log.Printf("ERROR: analyzer rejected contact reply for %s with status %d: %s", id, resp.StatusCode, body)
		writeError(w, http.StatusBadGateway, "Unable to send reply")
		return
	}

	updated, err := c.adminService.UpdateContactRequestStatus(r.Context(), id, "replied")
	if err != nil {
		log.Printf("ERROR: reply sent but failed to mark contact request %s as replied: %v", id, err)
		writeError(w, http.StatusInternalServerError, "Reply sent but status update failed")
		return
	}
	writeJSON(w, http.StatusOK, updated)
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
