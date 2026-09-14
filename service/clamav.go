package service

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ClamAVScanner wraps the ClamAV virus scanner.
// If ClamAV is not installed or not operational on the system, it operates in a "disabled" mode
// and skips scanning with a warning log.
type ClamAVScanner struct {
	available bool   // true if clamdscan or clamscan is found and runnable on the system
	command   string // the scanner binary to use
}

// isRunnable checks if the binary can be executed without dynamic linker or startup errors.
func isRunnable(bin string) bool {
	cmd := exec.Command(bin, "--version")
	return cmd.Run() == nil
}

// NewClamAVScanner checks if ClamAV is installed and returns a scanner.
// It prefers clamdscan (uses the clamd daemon — much faster for repeated scans).
// Fall back to clamscan (standalone — slower but no daemon needed).
func NewClamAVScanner() *ClamAVScanner {
	scanner := &ClamAVScanner{}

	// Allow explicitly disabling ClamAV via environment variable
	if enabled := os.Getenv("CLAMAV_ENABLED"); strings.ToLower(strings.TrimSpace(enabled)) == "false" || enabled == "0" {
		log.Println("ClamAV: Virus scanning is explicitly DISABLED via CLAMAV_ENABLED=false")
		scanner.available = false
		return scanner
	}

	// Prefer clamdscan (uses the clamd daemon — much faster for repeated scans).
	// Fall back to clamscan (standalone — slower but no daemon needed).
	if path, err := exec.LookPath("clamdscan"); err == nil && isRunnable(path) {
		scanner.available = true
		scanner.command = path
		log.Println("ClamAV: using clamdscan (daemon mode)")
	} else if path, err := exec.LookPath("clamscan"); err == nil && isRunnable(path) {
		scanner.available = true
		scanner.command = path
		log.Println("ClamAV: using clamscan (standalone mode — consider installing clamd for faster scans)")
	} else {
		scanner.available = false
		log.Println("WARNING: ClamAV not found or not operational on this system (missing libraries or daemon). Virus scanning is DISABLED.")
		log.Println("  To enable, ensure clamav is installed and libraries are present.")
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
// If ClamAV is not installed or operational, it logs a warning and returns nil (skips scan).
func (s *ClamAVScanner) ScanBytes(data []byte) error {
	if !s.available {
		log.Println("ClamAV: skipping scan (not operational or disabled)")
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
				// Exit code 2 = scanner error (e.g., daemon not running)
				isProd := strings.ToLower(os.Getenv("ENVIRONMENT")) == "production" ||
					strings.ToLower(os.Getenv("APP_ENV")) == "production"
				if !isProd {
					log.Printf("ClamAV: scanner error in development mode (%s): %v. Permitting upload.", outputStr, err)
					return nil
				}
				return fmt.Errorf("scan error: %s", outputStr)
			}
		}

		// Tool failure (e.g. exit code 127 due to missing shared libraries)
		isProd := strings.ToLower(os.Getenv("ENVIRONMENT")) == "production" ||
			strings.ToLower(os.Getenv("APP_ENV")) == "production"
		if !isProd {
			log.Printf("ClamAV: failed to run virus scan in development mode (%v: %s). Disabling scanner and permitting upload.", err, outputStr)
			s.available = false
			return nil
		}
		return fmt.Errorf("failed to run virus scan: %w", err)
	}

	// Exit code 0 = file is clean
	return nil
}
