package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type analysisJobRecord struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CVVersion      int
	FilePath       string
	FileName       string
	GithubUsername string
	Attempts       int
	MaxAttempts    int
}

// CVAnalysisWorker processes CV analysis jobs asynchronously from PostgreSQL.
type CVAnalysisWorker struct {
	pool        *pgxpool.Pool
	client      *AnalyzerClient
	uploadDir   string
	concurrency int
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewCVAnalysisWorker instantiates a new bounded worker queue.
func NewCVAnalysisWorker(pool *pgxpool.Pool, client *AnalyzerClient, uploadDir string, concurrency int) *CVAnalysisWorker {
	if concurrency <= 0 {
		concurrency = 2
	}
	if uploadDir == "" {
		uploadDir = CVUploadDir
	}
	return &CVAnalysisWorker{
		pool:        pool,
		client:      client,
		uploadDir:   uploadDir,
		concurrency: concurrency,
		stopChan:    make(chan struct{}),
	}
}

// Start launches the worker pool and stale-job recovery goroutines.
func (w *CVAnalysisWorker) Start(ctx context.Context) {
	log.Printf("Starting CV Analysis Worker pool (%d workers)", w.concurrency)

	for i := 1; i <= w.concurrency; i++ {
		w.wg.Add(1)
		go w.workerLoop(ctx, i)
	}

	w.wg.Add(1)
	go w.staleRecoveryLoop(ctx)
}

// Stop signals all workers to stop and waits for in-flight jobs to complete.
func (w *CVAnalysisWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	log.Println("CV Analysis Worker pool stopped successfully")
}

func (w *CVAnalysisWorker) workerLoop(ctx context.Context, workerID int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			job, found, err := w.claimNextJob(ctx)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}
			if !found {
				time.Sleep(2 * time.Second)
				continue
			}

			w.processJob(ctx, workerID, job)
		}
	}
}

// claimNextJob atomically claims one pending job using FOR UPDATE SKIP LOCKED.
func (w *CVAnalysisWorker) claimNextJob(ctx context.Context) (*analysisJobRecord, bool, error) {
	query := `
		UPDATE cv_analysis_jobs
		SET status = 'processing', started_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM cv_analysis_jobs
			WHERE status = 'pending'
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, user_id, cv_version, file_path, file_name, github_username, attempts, max_attempts;
	`

	var job analysisJobRecord
	err := w.pool.QueryRow(ctx, query).Scan(
		&job.ID,
		&job.UserID,
		&job.CVVersion,
		&job.FilePath,
		&job.FileName,
		&job.GithubUsername,
		&job.Attempts,
		&job.MaxAttempts,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &job, true, nil
}

func (w *CVAnalysisWorker) processJob(ctx context.Context, workerID int, job *analysisJobRecord) {
	log.Printf("[Worker %d] Processing analysis job %s for user %s (version %d)",
		workerID, job.ID, job.UserID, job.CVVersion)

	// 1. Security check: Validate file path is strictly within the trusted upload directory (prevent symlink attacks)
	realUploadDir, err := filepath.EvalSymlinks(w.uploadDir)
	if err != nil {
		realUploadDir = filepath.Clean(w.uploadDir)
	} else {
		realUploadDir = filepath.Clean(realUploadDir)
	}

	realFilePath, err := filepath.EvalSymlinks(job.FilePath)
	if err != nil {
		w.recordJobError(ctx, job.ID, job.Attempts, job.MaxAttempts, fmt.Sprintf("target file not found or invalid: %v", err))
		return
	}
	realFilePath = filepath.Clean(realFilePath)

	rel, err := filepath.Rel(realUploadDir, realFilePath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		errMsg := "rejected: file path resolved outside trusted upload directory (possible symlink attack)"
		log.Printf("[Worker %d] SECURITY ERROR on job %s: %s (%s)", workerID, job.ID, errMsg, job.FilePath)
		w.markJobFailed(ctx, job.ID, errMsg)
		return
	}

	// 2. Read file bytes from disk
	fileBytes, err := os.ReadFile(realFilePath)
	if err != nil {
		w.recordJobError(ctx, job.ID, job.Attempts, job.MaxAttempts, fmt.Sprintf("failed to read file from disk: %v", err))
		return
	}

	// 3. Call Python Analyzer directly
	res, err := w.client.AnalyzeCV(ctx, fileBytes, job.FileName, job.GithubUsername)
	if err != nil {
		log.Printf("[Worker %d] Analyzer failed for job %s: %v", workerID, job.ID, err)
		w.recordJobError(ctx, job.ID, job.Attempts, job.MaxAttempts, err.Error())
		return
	}

	// 4. Persist AI Summary in database idempotently (upsert on user_id, cv_version)
	insertSummaryQuery := `
		INSERT INTO ai_summaries (user_id, cv_version, summary, strengths, weaknesses, model, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, cv_version) WHERE cv_version IS NOT NULL
		DO UPDATE SET
			summary = EXCLUDED.summary,
			strengths = EXCLUDED.strengths,
			weaknesses = EXCLUDED.weaknesses,
			model = EXCLUDED.model,
			created_at = NOW()
	`
	_, err = w.pool.Exec(ctx, insertSummaryQuery,
		job.UserID,
		job.CVVersion,
		res.RawBody,
		res.Strengths,
		res.Weaknesses,
		res.Engine,
	)
	if err != nil {
		log.Printf("[Worker %d] Failed to save ai_summary for job %s: %v", workerID, job.ID, err)
		w.recordJobError(ctx, job.ID, job.Attempts, job.MaxAttempts, fmt.Sprintf("failed to save ai_summary: %v", err))
		return
	}

	// 5. Mark job completed
	updateQuery := `
		UPDATE cv_analysis_jobs
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, _ = w.pool.Exec(ctx, updateQuery, job.ID)
	log.Printf("[Worker %d] Successfully completed analysis job %s for user %s", workerID, job.ID, job.UserID)
}

func (w *CVAnalysisWorker) recordJobError(ctx context.Context, jobID uuid.UUID, currentAttempts, maxAttempts int, errMsg string) {
	newAttempts := currentAttempts + 1
	newStatus := "pending"
	if newAttempts >= maxAttempts {
		newStatus = "failed"
	}

	query := `
		UPDATE cv_analysis_jobs
		SET attempts = $2,
			last_error = $3,
			status = $4,
			updated_at = NOW()
		WHERE id = $1
	`
	_, _ = w.pool.Exec(ctx, query, jobID, newAttempts, errMsg, newStatus)
}

func (w *CVAnalysisWorker) markJobFailed(ctx context.Context, jobID uuid.UUID, errMsg string) {
	query := `
		UPDATE cv_analysis_jobs
		SET status = 'failed',
			last_error = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	_, _ = w.pool.Exec(ctx, query, jobID, errMsg)
}

// staleRecoveryLoop recovers jobs stuck in 'processing' status due to crashes or timeouts.
func (w *CVAnalysisWorker) staleRecoveryLoop(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recoverStaleJobs(ctx)
		}
	}
}

func (w *CVAnalysisWorker) recoverStaleJobs(ctx context.Context) {
	// Re-queue processing jobs that have been stuck for over 10 minutes (subject to retry budget)
	requeueQuery := `
		UPDATE cv_analysis_jobs
		SET status = 'pending',
			attempts = attempts + 1,
			last_error = 'Stale processing timeout: returned to pending queue',
			updated_at = NOW()
		WHERE status = 'processing'
		  AND started_at < NOW() - INTERVAL '10 minutes'
		  AND attempts + 1 < max_attempts;
	`
	cmdTag, err := w.pool.Exec(ctx, requeueQuery)
	if err == nil && cmdTag.RowsAffected() > 0 {
		log.Printf("Recovered %d stale processing CV analysis jobs back to pending", cmdTag.RowsAffected())
	}

	// Permanently fail jobs that exceeded max_attempts
	failQuery := `
		UPDATE cv_analysis_jobs
		SET status = 'failed',
			last_error = 'Stale processing timeout: exceeded max retry attempts',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE status = 'processing'
		  AND started_at < NOW() - INTERVAL '10 minutes'
		  AND attempts + 1 >= max_attempts;
	`
	_, _ = w.pool.Exec(ctx, failQuery)
}
