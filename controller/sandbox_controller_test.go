package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
)

func TestSandboxController_Execute_RejectsOversizedCode(t *testing.T) {
	c := NewSandboxController(service.NewSandboxService())

	// Create code larger than 64KB
	hugeCode := strings.Repeat("x = 1\n", 12000)
	reqBody, err := json.Marshal(models.ExecuteRequest{
		Language: "python",
		Code:     hugeCode,
		Type:     models.ExecutionTypeStandard,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sandbox/execute", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	c.Execute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for oversized code, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "exceeds maximum limit") {
		t.Fatalf("expected body to mention maximum limit, got: %s", w.Body.String())
	}
}

func TestSandboxController_ParseCode_RejectsOversizedCode(t *testing.T) {
	c := NewSandboxController(service.NewSandboxService())

	hugeCode := strings.Repeat("x = 1\n", 12000)
	reqBody, err := json.Marshal(models.ParseRequest{
		Language: "python",
		Code:     hugeCode,
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sandbox/parse", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	c.ParseCode(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for oversized parse code, got %d", w.Code)
	}
}
