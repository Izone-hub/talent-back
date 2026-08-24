package controller

import (
	"encoding/json"
	"net/http"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
)

type SandboxController struct {
	sandboxService *service.SandboxService
}

func NewSandboxController(sandboxService *service.SandboxService) *SandboxController {
	return &SandboxController{
		sandboxService: sandboxService,
	}
}

func (c *SandboxController) Execute(w http.ResponseWriter, r *http.Request) {
	var req models.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Language == "" {
		writeError(w, http.StatusBadRequest, "language is required")
		return
	}
	if req.Code == "" && req.Type != models.ExecutionTypeFramework {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	result, err := c.sandboxService.Execute(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (c *SandboxController) ListLanguages(w http.ResponseWriter, r *http.Request) {
	languages := []map[string]string{
		{"id": "python", "name": "Python", "type": "standard"},
		{"id": "javascript", "name": "JavaScript", "type": "standard"},
		{"id": "go", "name": "Go", "type": "standard"},
		{"id": "java", "name": "Java", "type": "standard"},
		{"id": "cpp", "name": "C++", "type": "standard"},
		{"id": "c", "name": "C", "type": "standard"},
		{"id": "rust", "name": "Rust", "type": "standard"},
		{"id": "ruby", "name": "Ruby", "type": "standard"},
		{"id": "typescript", "name": "TypeScript", "type": "standard"},
		{"id": "dart", "name": "Dart", "type": "standard"},
		{"id": "sql", "name": "SQL (SQLite)", "type": "standard"},

		{"id": "react", "name": "React", "type": "framework"},
		{"id": "vue", "name": "Vue", "type": "framework"},
		{"id": "svelte", "name": "Svelte", "type": "framework"},
		{"id": "express", "name": "Node.js / Express", "type": "framework"},
		{"id": "nextjs", "name": "Next.js", "type": "framework"},
		{"id": "flutter", "name": "Flutter", "type": "framework"},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"languages": languages,
	})
}

func (c *SandboxController) ParseCode(w http.ResponseWriter, r *http.Request) {
	var req models.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Language == "" {
		writeError(w, http.StatusBadRequest, "language is required")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	result, err := c.sandboxService.ParseCode(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
