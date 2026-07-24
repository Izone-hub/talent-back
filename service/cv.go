package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/Izone-hub/talent-backend/models"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	MaxCVSize       = 5 << 20 // 5 MB
	CVUploadDir     = "uploads/cv"
	MaxUploadsPerHr = 5 // rate limit: uploads per user per hour
)

// pdfMagicBytes are the first 5 bytes of every valid PDF file.
var pdfMagicBytes = []byte{0x25, 0x50, 0x44, 0x46, 0x2D} // %PDF-

// ---------------------------------------------------------------------------
// Rate Limiter (in-memory, per-user)
// ---------------------------------------------------------------------------

type uploadEntry struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[uuid.UUID]*uploadEntry
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		entries: make(map[uuid.UUID]*uploadEntry),
	}
}

// allow checks whether the user is within rate limits.
// Returns true if the upload is allowed, false if rate-limited.
func (rl *rateLimiter) allow(userID uuid.UUID) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[userID]

	if !exists || now.Sub(entry.windowStart) > time.Hour {
		// New window
		rl.entries[userID] = &uploadEntry{count: 1, windowStart: now}
		return true
	}

	if entry.count >= MaxUploadsPerHr {
		return false
	}

	entry.count++
	return true
}

// ---------------------------------------------------------------------------
// CV Service
// ---------------------------------------------------------------------------

// CvService handles all CV upload and management logic.
type CvService struct {
	queries     *database.Queries
	scanner     *ClamAVScanner
	rateLimiter *rateLimiter
	uploadDir   string // base directory for CV storage
}

// NewCvService creates a new CvService.
func NewCvService(db database.DBTX, scanner *ClamAVScanner) *CvService {
	// Ensure upload directory exists
	if err := os.MkdirAll(CVUploadDir, 0755); err != nil {
		log.Printf("WARNING: could not create upload dir %s: %v", CVUploadDir, err)
	}

	return &CvService{
		queries:     database.New(db),
		scanner:     scanner,
		rateLimiter: newRateLimiter(),
		uploadDir:   CVUploadDir,
	}
}

// ---------------------------------------------------------------------------
// Upload Pipeline
// ---------------------------------------------------------------------------

// UploadCV runs the full 7-step upload pipeline:
//  1. Rate limit check
//  2. Format validation (extension + magic bytes)
//  3. Size validation (already enforced by MaxBytesReader, double-check here)
//  4. Hash & deduplicate
//  5. Virus scan
//  6. Save to disk
//  7. Save DB record & return response
func (s *CvService) UploadCV(ctx context.Context, userID uuid.UUID, fileName string, fileBytes []byte, clientIP string) (*models.CvVersion, error) {

	// ── Step 1: Rate limit ──────────────────────────────────────────────
	if !s.rateLimiter.allow(userID) {
		return nil, fmt.Errorf("rate limit exceeded: max %d uploads per hour", MaxUploadsPerHr)
	}

	// ── Step 2: Format validation ───────────────────────────────────────
	// 2a. Check file extension
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		return nil, fmt.Errorf("only PDF files are accepted")
	}

	// 2b. Check magic bytes (prevents renamed .exe → .pdf attacks)
	if len(fileBytes) < 5 {
		return nil, fmt.Errorf("file is too small to be a valid PDF")
	}
	for i, b := range pdfMagicBytes {
		if fileBytes[i] != b {
			return nil, fmt.Errorf("file content is not a valid PDF (magic bytes mismatch)")
		}
	}

	// ── Step 3: Size validation (defense in depth) ──────────────────────
	if len(fileBytes) > MaxCVSize {
		return nil, fmt.Errorf("file size %d bytes exceeds maximum %d bytes (5 MB)", len(fileBytes), MaxCVSize)
	}

	// ── Step 4: Hash & deduplicate ──────────────────────────────────────
	hash := sha256.Sum256(fileBytes)
	fileHash := hex.EncodeToString(hash[:])

	// Check if the exact same file was already uploaded by this user
	existingCV, err := s.queries.FindDuplicateCV(ctx, database.FindDuplicateCVParams{
		UserID:   uuidToPgUUID(userID),
		FileHash: fileHash,
	})
	if err == nil {
		// Duplicate found — re-mark it as the current CV and return it.
		// The existing record may have is_current=false if a newer CV was
		// uploaded after it, so we need to reset the flag.
		pgUserID := uuidToPgUUID(userID)
		if err := s.queries.SetCurrentCV(ctx, pgUserID); err != nil {
			log.Printf("WARNING: failed to unset current CV during dedup: %v", err)
		}
		if err := s.queries.MarkCVAsCurrent(ctx, database.MarkCVAsCurrentParams{
			ID:     existingCV.ID,
			UserID: pgUserID,
		}); err != nil {
			log.Printf("WARNING: failed to re-mark duplicate CV as current: %v", err)
		}
		// Re-fetch the CV to get the updated is_current flag
		existingCV.IsCurrent.Bool = true
		existingCV.IsCurrent.Valid = true
		cv := dbCvToModel(existingCV)
		return &cv, nil
	}
	// err != nil means no duplicate found (pgx.ErrNoRows) — continue

	// ── Step 5: Virus scan ──────────────────────────────────────────────
	if err := s.scanner.ScanBytes(fileBytes); err != nil {
		return nil, fmt.Errorf("virus scan failed: %w", err)
	}

	// ── Step 6: Save to disk ────────────────────────────────────────────
	// Directory: uploads/cv/{user_id}/
	userDir := filepath.Join(s.uploadDir, userID.String())
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Filename: {uuid}.pdf (prevents collisions and path traversal)
	diskFileName := uuid.New().String() + ".pdf"
	diskPath := filepath.Join(userDir, diskFileName)

	if err := os.WriteFile(diskPath, fileBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file to disk: %w", err)
	}

	// ── Step 7: Database record ─────────────────────────────────────────
	// 7a. Mark all previous CVs as not current
	pgUserID := uuidToPgUUID(userID)
	if err := s.queries.SetCurrentCV(ctx, pgUserID); err != nil {
		log.Printf("WARNING: failed to unset current CV: %v", err)
	}

	// 7b. Parse client IP for the database
	var ipAddr *netip.Addr
	if clientIP != "" {
		// Strip port if present (e.g., "192.168.1.1:54321" → "192.168.1.1")
		host := clientIP
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			host = clientIP[:idx]
		}
		if parsed, err := netip.ParseAddr(host); err == nil {
			ipAddr = &parsed
		}
	}

	// 7c. Insert the new CV version
	dbCV, err := s.queries.CreateCV(ctx, database.CreateCVParams{
		UserID:         pgUserID,
		FileName:       fileName,
		FilePath:       diskPath,
		FileSize:       int32(len(fileBytes)),
		FileHash:       fileHash,
		UploadedFromIp: ipAddr,
		// ApplicationID:  uuid.Nil, // not linked to an application yet
	})
	if err != nil {
		// Clean up the file we just saved
		os.Remove(diskPath)
		return nil, fmt.Errorf("failed to save CV record: %w", err)
	}

	// 7d. Clean up old versions (keep last 5)
	_ = s.queries.DeleteOldCVs(ctx, pgUserID)

	cv := dbCvToModel(dbCV)
	return &cv, nil
}

// ---------------------------------------------------------------------------
// Read operations
// ---------------------------------------------------------------------------

// GetCurrentCV returns the user's current (latest) CV.
func (s *CvService) GetCurrentCV(ctx context.Context, userID uuid.UUID) (*models.CvVersion, error) {
	dbCV, err := s.queries.GetCurrentCV(ctx, uuidToPgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("no current CV found: %w", err)
	}
	cv := dbCvToModel(dbCV)
	return &cv, nil
}

// ListCVVersions returns all CV versions for a user.
func (s *CvService) ListCVVersions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.CvVersion, error) {
	dbCVs, err := s.queries.ListCVsByUser(ctx, database.ListCVsByUserParams{
		UserID: uuidToPgUUID(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list CVs: %w", err)
	}

	cvs := make([]models.CvVersion, 0, len(dbCVs))
	for _, dbCV := range dbCVs {
		cvs = append(cvs, dbCvToModel(dbCV))
	}
	return cvs, nil
}

// DeleteCV deletes a specific CV version (file + DB record).
func (s *CvService) DeleteCV(ctx context.Context, userID, cvID uuid.UUID) error {
	// Get the CV to find the file path
	dbCV, err := s.queries.GetCVByID(ctx, uuidToPgUUID(cvID))
	if err != nil {
		return fmt.Errorf("CV not found: %w", err)
	}

	// Verify ownership
	var ownerID uuid.UUID
	if dbCV.UserID.Valid {
		ownerID, _ = uuid.FromBytes(dbCV.UserID.Bytes[:])
	}
	if ownerID != userID {
		return fmt.Errorf("you do not own this CV")
	}

	// Delete from DB first
	if err := s.queries.DeleteCV(ctx, database.DeleteCVParams{
		ID:     uuidToPgUUID(cvID),
		UserID: uuidToPgUUID(userID),
	}); err != nil {
		return fmt.Errorf("failed to delete CV record: %w", err)
	}

	// Delete file from disk (best-effort)
	if err := os.Remove(dbCV.FilePath); err != nil {
		log.Printf("WARNING: failed to delete CV file %s: %v", dbCV.FilePath, err)
	}

	return nil
}

// DownloadCV returns the file path for a CV (used by the controller to serve the file).
func (s *CvService) DownloadCV(ctx context.Context, userID, cvID uuid.UUID) (filePath, fileName string, err error) {
	dbCV, err := s.queries.GetCVByID(ctx, uuidToPgUUID(cvID))
	if err != nil {
		return "", "", fmt.Errorf("CV not found: %w", err)
	}

	// Verify ownership
	var ownerID uuid.UUID
	if dbCV.UserID.Valid {
		ownerID, _ = uuid.FromBytes(dbCV.UserID.Bytes[:])
	}
	if ownerID != userID {
		return "", "", fmt.Errorf("you do not own this CV")
	}

	return dbCV.FilePath, dbCV.FileName, nil
}

// ---------------------------------------------------------------------------
// Conversion helper: database.CvVersion → models.CvVersion
// ---------------------------------------------------------------------------

func dbCvToModel(cv database.CvVersion) models.CvVersion {
	var id uuid.UUID
	if cv.ID.Valid {
		id, _ = uuid.FromBytes(cv.ID.Bytes[:])
	}
	var userID uuid.UUID
	if cv.UserID.Valid {
		userID, _ = uuid.FromBytes(cv.UserID.Bytes[:])
	}
	var appID *uuid.UUID
	if cv.ApplicationID != uuid.Nil {
		appID = &cv.ApplicationID
	}
	var ip *string
	if cv.UploadedFromIp != nil {
		s := cv.UploadedFromIp.String()
		ip = &s
	}

	return models.CvVersion{
		ID:             id,
		UserID:         userID,
		FileName:       cv.FileName,
		FilePath:       cv.FilePath,
		FileSize:       int(cv.FileSize),
		FileHash:       cv.FileHash,
		Version:        int(cv.Version),
		IsCurrent:      cv.IsCurrent.Bool,
		UploadedAt:     pgTimestampToTime(cv.UploadedAt),
		UploadedFromIP: ip,
		ApplicationID:  appID,
		CreatedAt:      pgTimestampToTime(cv.CreatedAt),
	}
}
