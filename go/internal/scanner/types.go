package scanner

import (
	"context"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// Scanner defines the interface for external security scanners
type Scanner interface {
	// Name returns the scanner name
	Name() string

	// IsInstalled checks if the scanner is available
	IsInstalled() bool

	// Version returns the scanner version if installed
	Version() string

	// Run executes the scanner and returns findings
	Run(ctx context.Context, scope string) ([]findings.Finding, error)

	// Type returns the scanner type (local or cloud)
	Type() ScannerType
}

// ScannerType represents the type of scanner
type ScannerType string

const (
	TypeLocal ScannerType = "local"
	TypeCloud ScannerType = "cloud"
)

// ScannerStatus represents the status of a scanner
type ScannerStatus struct {
	Name      string
	Type      ScannerType
	Installed bool
	Version   string
	Enabled   bool
	Skipped   bool
	Reason    string // Why it was skipped
	Ran       bool   // Whether it ran successfully
	Found     int    // Number of findings
	Error     error  // Error if scan failed
}

// ScanResult contains results from a scanner run
type ScanResult struct {
	Scanner  string
	Findings []findings.Finding
	Error    error
}
