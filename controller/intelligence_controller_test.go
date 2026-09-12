package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Izone-hub/talent-backend/service"
)

func TestIntelligenceController_GetLatestUserSummary_WhenQueriesMissing(t *testing.T) {
	c := &IntelligenceController{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/intelligence/11111111-1111-1111-1111-111111111111/summary", nil)
	req.SetPathValue("id", "11111111-1111-1111-1111-111111111111")
	ctx := context.WithValue(req.Context(), "user", &service.Claims{
		Role: "admin",
	})
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()

	c.GetLatestUserSummary(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal server error when DB is not configured, got %d", res.Code)
	}
}
