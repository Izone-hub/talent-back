package service

import (
	"os"
	"testing"
)

func TestClamAVScanner_Disabled(t *testing.T) {
	os.Setenv("CLAMAV_ENABLED", "false")
	defer os.Unsetenv("CLAMAV_ENABLED")

	scanner := NewClamAVScanner()
	if scanner.IsAvailable() {
		t.Errorf("expected scanner to be disabled when CLAMAV_ENABLED=false")
	}

	err := scanner.ScanBytes([]byte("fake cv data"))
	if err != nil {
		t.Errorf("expected ScanBytes to return nil when disabled, got %v", err)
	}
}

func TestClamAVScanner_BrokenBinaryGracefulFallback(t *testing.T) {
	os.Unsetenv("CLAMAV_ENABLED")

	scanner := NewClamAVScanner()
	if scanner.IsAvailable() {
		t.Errorf("expected scanner to not be available due to broken shared libraries")
	}

	err := scanner.ScanBytes([]byte("%PDF-1.4 sample content"))
	if err != nil {
		t.Errorf("expected ScanBytes to not return error when scanner unavailable, got %v", err)
	}
}
