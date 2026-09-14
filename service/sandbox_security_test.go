package service

import (
	"context"
	"testing"
	"time"
)

func TestSandbox_ConcurrencyLimiter(t *testing.T) {
	var releases []func()
	defer func() {
		for _, r := range releases {
			r()
		}
	}()

	// Acquire all 8 slots
	for i := 0; i < 8; i++ {
		release, err := acquireSandboxSlot(context.Background())
		if err != nil {
			t.Fatalf("failed to acquire slot %d: %v", i, err)
		}
		releases = append(releases, release)
	}

	// 9th acquisition with short deadline should fail
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := acquireSandboxSlot(ctx)
	if err == nil {
		t.Fatalf("expected 9th slot acquisition to fail when capacity is reached, but it succeeded")
	}

	// Release one slot
	releases[0]()
	releases = releases[1:]

	// Now acquisition should succeed
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()

	rel, err := acquireSandboxSlot(ctx2)
	if err != nil {
		t.Fatalf("expected slot acquisition to succeed after releasing a slot, got: %v", err)
	}
	releases = append(releases, rel)
}
