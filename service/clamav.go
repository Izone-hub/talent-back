package service

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// ClamAVScanner wraps the ClamAV virus scanner.
// If ClamAV is not installed on the system, it operates in a "disabled" mode
// and skips scanning with a warning log.
type ClamAVScanner struct {
	available bool   // true if clamdscan or clamscan is found on the system
	command   string // the scanner binary to use
}

// NewClamAVScanner checks if ClamAV is installed and returns a scanner.
// It prefers clamdscan (daemon mode, faster) over clamscan (standalone).
func NewClamAVScanner() *ClamAVScanner {
	scanner := &ClamAVScanner{}

	// Prefer clamdscan (uses the clamd daemon — much faster for repeated scans).
	// Fall back to clamscan (standalone — slower but no daemon needed).
	if path, err := exec.LookPath("clamdscan"); err == nil {
		scanner.available = true
		scanner.command = path
		log.Println("ClamAV: using clamdscan (daemon mode)")
	} else if path, err := exec.LookPath("clamscan"); err == nil {
		scanner.available = true
		scanner.command = path
		log.Println("ClamAV: using clamscan (standalone mode — consider installing clamd for faster scans)")
	} else {
		scanner.available = false
		log.Println("WARNING: ClamAV not found on this system. Virus scanning is DISABLED.")
		log.Println("  Install with: sudo apt install clamav clamav-daemon")
	}

	return scanner
}

// IsAvailable returns whether ClamAV is installed and usable.
func (s *ClamAVScanner) IsAvailable() bool {
	return s.available
}

// ScanBytes scans the given file content for viruses.
// Returns nil if the file is clean, or an error describing the threat.
//
// If ClamAV is not installed, it logs a warning and returns nil (skips scan).
//
// How it works:
//   - Pipes the file bytes to clamdscan/clamscan via stdin (--no-summary -)
//   - Exit code 0 = clean, exit code 1 = virus found, exit code 2 = error
//   - On virus detection, the output contains the threat name
func (s *ClamAVScanner) ScanBytes(data []byte) error {
	if !s.available {
		log.Println("ClamAV: skipping scan (not installed)")
		return nil
	}

	// Run: clamdscan --no-summary -
	// The "-" at the end tells it to read from stdin
	cmd := exec.Command(s.command, "--no-summary", "-")
	cmd.Stdin = bytes.NewReader(data)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		// Check if it's an exit code error
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 1:
				// Exit code 1 = virus found
				return fmt.Errorf("virus detected: %s", outputStr)
			case 2:
				// Exit code 2 = scanner error (e.g., corrupted file)
				return fmt.Errorf("scan error: %s", outputStr)
			}
		}
		return fmt.Errorf("failed to run virus scan: %w", err)
	}

	// Exit code 0 = file is clean
	return nil
}
