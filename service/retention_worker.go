package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultAuditRetentionDays          = 365
	DefaultCVJobCompletedRetentionDays = 14
	DefaultCVJobFailedRetentionDays    = 90
	DefaultBatchSize                   = 500
	DefaultBatchPause                  = 50 * time.Millisecond
)

// RetentionRunResult reports row count metrics from a single retention sweep.
type RetentionRunResult struct {
	AuditLogsPurged       int
	CompletedCVJobsPurged int
	FailedCVJobsPurged    int
	Duration              time.Duration
}

// RetentionWorker periodically cleans up historical and queue records in batches.
type RetentionWorker struct {
	pool                       *pgxpool.Pool
	interval                   time.Duration
	batchSize                  int
	batchPause                 time.Duration
	auditRetentionDays         int
	cvJobCompletedRetentionDays int
	cvJobFailedRetentionDays   int
	stopChan                   chan struct{}
	wg                         sync.WaitGroup
}

// NewRetentionWorker creates an instance of RetentionWorker.
func NewRetentionWorker(pool *pgxpool.Pool, interval time.Duration) *RetentionWorker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &RetentionWorker{
		pool:                       pool,
		interval:                   interval,
		batchSize:                  DefaultBatchSize,
		batchPause:                 DefaultBatchPause,
		auditRetentionDays:         DefaultAuditRetentionDays,
		cvJobCompletedRetentionDays: DefaultCVJobCompletedRetentionDays,
		cvJobFailedRetentionDays:   DefaultCVJobFailedRetentionDays,
		stopChan:                   make(chan struct{}),
	}
}

// Start launches the background retention loop.
func (w *RetentionWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		log.Printf("[RetentionWorker] Starting background retention worker (interval=%v, batchSize=%d)", w.interval, w.batchSize)

		// Initial run on startup
		if res, err := w.RunOnce(ctx); err != nil {
			log.Printf("[RetentionWorker] Initial retention sweep encountered error: %v", err)
		} else {
			log.Printf("[RetentionWorker] Initial sweep completed in %v: purged %d audit logs, %d completed CV jobs, %d failed CV jobs",
				res.Duration, res.AuditLogsPurged, res.CompletedCVJobsPurged, res.FailedCVJobsPurged)
		}

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if res, err := w.RunOnce(ctx); err != nil {
					log.Printf("[RetentionWorker] Periodic retention sweep encountered error: %v", err)
				} else {
					log.Printf("[RetentionWorker] Sweep completed in %v: purged %d audit logs, %d completed CV jobs, %d failed CV jobs",
						res.Duration, res.AuditLogsPurged, res.CompletedCVJobsPurged, res.FailedCVJobsPurged)
				}
			case <-w.stopChan:
				log.Println("[RetentionWorker] Stopped background retention worker")
				return
			case <-ctx.Done():
				log.Println("[RetentionWorker] Context canceled, stopping retention worker")
				return
			}
		}
	}()
}

// Stop gracefully stops the worker.
func (w *RetentionWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// RunOnce performs a complete batched cleanup sweep across targeted tables.
func (w *RetentionWorker) RunOnce(ctx context.Context) (RetentionRunResult, error) {
	start := time.Now()
	var result RetentionRunResult

	// 1. Purge audit_logs in small batches
	auditPurged, err := w.purgeAuditLogs(ctx)
	if err != nil {
		log.Printf("[RetentionWorker] Error purging audit_logs: %v", err)
	}
	result.AuditLogsPurged = auditPurged

	// 2. Purge completed cv_analysis_jobs in small batches
	completedJobsPurged, err := w.purgeCompletedCVJobs(ctx)
	if err != nil {
		log.Printf("[RetentionWorker] Error purging completed cv_analysis_jobs: %v", err)
	}
	result.CompletedCVJobsPurged = completedJobsPurged

	// 3. Purge old failed cv_analysis_jobs in small batches
	failedJobsPurged, err := w.purgeFailedCVJobs(ctx)
	if err != nil {
		log.Printf("[RetentionWorker] Error purging failed cv_analysis_jobs: %v", err)
	}
	result.FailedCVJobsPurged = failedJobsPurged

	result.Duration = time.Since(start)
	return result, nil
}

// purgeAuditLogs deletes audit logs older than the retention threshold using batched DELETEs.
func (w *RetentionWorker) purgeAuditLogs(ctx context.Context) (int, error) {
	query := `
		DELETE FROM audit_logs
		WHERE id IN (
			SELECT id FROM audit_logs
			WHERE performed_at < NOW() - make_interval(days => $1)
			LIMIT $2
		);
	`
	return w.executeBatchedDelete(ctx, "audit_logs", query, w.auditRetentionDays)
}

// purgeCompletedCVJobs deletes completed CV analysis jobs older than the retention threshold in batches.
// Never deletes pending or processing jobs.
func (w *RetentionWorker) purgeCompletedCVJobs(ctx context.Context) (int, error) {
	query := `
		DELETE FROM cv_analysis_jobs
		WHERE id IN (
			SELECT id FROM cv_analysis_jobs
			WHERE status = 'completed'
			  AND completed_at IS NOT NULL
			  AND completed_at < NOW() - make_interval(days => $1)
			LIMIT $2
		);
	`
	return w.executeBatchedDelete(ctx, "completed cv_analysis_jobs", query, w.cvJobCompletedRetentionDays)
}

// purgeFailedCVJobs deletes old failed CV analysis jobs in batches for troubleshooting retention.
func (w *RetentionWorker) purgeFailedCVJobs(ctx context.Context) (int, error) {
	query := `
		DELETE FROM cv_analysis_jobs
		WHERE id IN (
			SELECT id FROM cv_analysis_jobs
			WHERE status = 'failed'
			  AND updated_at < NOW() - make_interval(days => $1)
			LIMIT $2
		);
	`
	return w.executeBatchedDelete(ctx, "failed cv_analysis_jobs", query, w.cvJobFailedRetentionDays)
}

// executeBatchedDelete runs a delete query in batches of batchSize until 0 rows remain or context is canceled.
func (w *RetentionWorker) executeBatchedDelete(ctx context.Context, label, query string, retentionDays int) (int, error) {
	totalDeleted := 0

	for {
		select {
		case <-ctx.Done():
			return totalDeleted, ctx.Err()
		case <-w.stopChan:
			return totalDeleted, nil
		default:
		}

		tag, err := w.pool.Exec(ctx, query, retentionDays, w.batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("batched delete for %s failed: %w", label, err)
		}

		deleted := int(tag.RowsAffected())
		totalDeleted += deleted

		// If fewer rows than batchSize were deleted, all matching rows have been purged
		if deleted < w.batchSize {
			break
		}

		// Pause between batches to avoid WAL spikes and give other transactions lock priority
		if w.batchPause > 0 {
			time.Sleep(w.batchPause)
		}
	}

	return totalDeleted, nil
}
