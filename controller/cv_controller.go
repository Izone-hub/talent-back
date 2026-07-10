package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Izone-hub/talent-backend/models"
	"github.com/Izone-hub/talent-backend/service"
	"github.com/google/uuid"
)

// CvController handles HTTP requests for CV upload and management.
type CvController struct {
	cvService *service.CvService
}

// NewCvController creates a new CvController.
func NewCvController(cvService *service.CvService) *CvController {
	return &CvController{
		cvService: cvService,
	}
}

// ---------------------------------------------------------------------------
// POST /cv/upload — Upload a CV (multipart/form-data)
// ---------------------------------------------------------------------------
//
// How multipart file upload works in Go:
//
//  1. http.MaxBytesReader wraps r.Body to cap the total request size.
//     If the client sends more than 5 MB, Read() returns an error immediately
//     — no memory wasted.
//
//  2. r.FormFile("cv") parses the multipart boundary, finds the part named "cv",
//     and returns:
//     - file   → an io.ReadCloser (the file data stream)
//     - header → metadata (original filename, declared size, MIME type)
//
//  3. We read ALL bytes into memory with io.ReadAll(file).
//     This is fine for 5 MB max. For larger files you'd stream directly to disk.
//
//  4. We pass the bytes to CvService.UploadCV() which runs the full pipeline.

func (c *CvController) UploadCV(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Step 1: Cap request body at 5 MB.
	// If the client sends more, r.Body.Read() will return an error.
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxCVSize)

	// Step 2: Get the file from the multipart form.
	// "cv" is the form field name the client must use:
	//   curl -F "cv=@resume.pdf" http://localhost:5000/api/v1/cv/upload
	file, header, err := r.FormFile("cv")
	if err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusRequestEntityTooLarge, "File too large. Maximum size is 5 MB.")
			return
		}
		writeError(w, http.StatusBadRequest, "Missing or invalid file. Use form field name 'cv'.")
		return
	}
	defer file.Close()

	// Step 3: Read all file bytes into memory.
	// Safe because MaxBytesReader already caps at 5 MB.
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusRequestEntityTooLarge, "File too large. Maximum size is 5 MB.")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	// Step 4: Get client IP for audit logging
	clientIP := r.RemoteAddr

	// Step 5: Run the full upload pipeline (validate → hash → scan → save → DB)
	cv, err := c.cvService.UploadCV(r.Context(), userID, header.Filename, fileBytes, clientIP)
	if err != nil {
		// Determine appropriate status code based on the error
		status := http.StatusBadRequest
		if contains(err.Error(), "virus") {
			status = http.StatusUnprocessableEntity
		} else if contains(err.Error(), "rate limit") {
			status = http.StatusTooManyRequests
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, cv.ToResponse())
}

func triggerCVAnalysis(filePath, fileName, githubUsername string) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	fw, err := mp.CreateFormFile("file", fileName)
	if err != nil {
		return
	}
	if _, err := fw.Write(fileBytes); err != nil {
		return
	}
	if githubUsername != "" {
		mp.WriteField("github_username", githubUsername)
	}
	mp.Close()

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post("http://localhost:5000/api/v1/analyze-cv", mp.FormDataContentType(), &buf)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	historyDir := "history"
	os.MkdirAll(historyDir, 0755)
	stem := filepath.Base(fileName)
	if ext := filepath.Ext(stem); ext != "" {
		stem = stem[:len(stem)-len(ext)]
	}
	ts := time.Now().Unix()
	historyPath := filepath.Join(historyDir, fmt.Sprintf("upload_%s_%d.json", stem, ts))
	os.WriteFile(historyPath, body, 0644)
}

// ---------------------------------------------------------------------------
// GET /cv/current — Get the user's current CV metadata
// ---------------------------------------------------------------------------

func (c *CvController) GetCurrentCV(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	cv, err := c.cvService.GetCurrentCV(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "No CV found")
		return
	}

	writeJSON(w, http.StatusOK, cv.ToResponse())
}

// ---------------------------------------------------------------------------
// GET /cv/versions — List all CV versions for the authenticated user
// ---------------------------------------------------------------------------

func (c *CvController) ListCVVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, offset := parsePagination(r)

	cvs, err := c.cvService.ListCVVersions(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]models.CvUploadResponse, 0, len(cvs))
	for _, cv := range cvs {
		responses = append(responses, cv.ToResponse())
	}

	writeJSON(w, http.StatusOK, models.CvVersionListResponse{
		Versions: responses,
		Total:    len(responses),
	})
}

// ---------------------------------------------------------------------------
// GET /cv/{id}/download — Download a specific CV file
// ---------------------------------------------------------------------------

func (c *CvController) DownloadCV(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	cvID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid CV ID")
		return
	}

	filePath, fileName, err := c.cvService.DownloadCV(r.Context(), userID, cvID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Serve the file with proper headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	http.ServeFile(w, r, filePath)
}

// ---------------------------------------------------------------------------
// DELETE /cv/{id} — Delete a specific CV version
// ---------------------------------------------------------------------------

func (c *CvController) DeleteCV(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	cvID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid CV ID")
		return
	}

	if err := c.cvService.DeleteCV(r.Context(), userID, cvID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "CV deleted successfully"})
}

// contains is a simple helper to check substring presence.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
