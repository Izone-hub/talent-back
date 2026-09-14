package service

import (
	"context"
	"testing"
	"time"
)

func TestRetentionWorkerLifecycle(t *testing.T) {
	worker := NewRetentionWorker(nil, 50*time.Millisecond)
	if worker.interval != 50*time.Millisecond {
		t.Fatalf("expected interval 50ms, got %v", worker.interval)
	}
	if worker.batchSize != DefaultBatchSize {
		t.Fatalf("expected batchSize %d, got %d", DefaultBatchSize, worker.batchSize)
	}
	if worker.auditRetentionDays != DefaultAuditRetentionDays {
		t.Fatalf("expected auditRetentionDays %d, got %d", DefaultAuditRetentionDays, worker.auditRetentionDays)
	}
	if worker.cvJobCompletedRetentionDays != DefaultCVJobCompletedRetentionDays {
		t.Fatalf("expected cvJobCompletedRetentionDays %d, got %d", DefaultCVJobCompletedRetentionDays, worker.cvJobCompletedRetentionDays)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Calling Stop on an idle worker should not block or panic
	worker.Stop()
	_ = ctx
}
