package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/liam-witterick/infra-review/go/internal/analysis"
	"github.com/liam-witterick/infra-review/go/internal/findings"
)

// CheckovScanner implements Scanner for Checkov
type CheckovScanner struct {
	binaryPath string
}

const (
	checkovFrameworks = "terraform,dockerfile,kubernetes,helm,serverless"
)

// NewCheckovScanner creates a new Checkov scanner
func NewCheckovScanner() *CheckovScanner {
	return &CheckovScanner{
		binaryPath: "checkov",
	}
}

// Name returns the scanner name
func (s *CheckovScanner) Name() string {
	return "checkov"
}

// Type returns the scanner type
func (s *CheckovScanner) Type() ScannerType {
	return TypeLocal
}

// IsInstalled checks if Checkov is available
func (s *CheckovScanner) IsInstalled() bool {
	_, err := exec.LookPath(s.binaryPath)
	return err == nil
}

// Version returns the Checkov version
func (s *CheckovScanner) Version() string {
	if !s.IsInstalled() {
		return ""
	}
	
	cmd := exec.Command(s.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "installed"
	}
	
	// Parse version from output
	version := strings.TrimSpace(string(output))
	return version
}

// Run executes Checkov and returns findings
func (s *CheckovScanner) Run(ctx context.Context, scope string) ([]findings.Finding, error) {
	// Run checkov with JSON output
	args := []string{
		"-d", ".",
		"--quiet",
		"--compact",
		"-o", "json",
		"--skip-download", // Don't download updates during scan
	}
	
	// Add framework filters based on scope
	if scope == "security" || scope == "all" {
		// Include all frameworks for comprehensive security scan
		args = append(args, "--framework", checkovFrameworks)
	}
	
	cmd := exec.CommandContext(ctx, s.binaryPath, args...)
	output, err := cmd.Output()
	
	// Checkov returns non-zero exit code when it finds issues, which is expected
	// We only care about parsing errors
	if err != nil && len(output) == 0 {
		// Real error - no output
		return nil, fmt.Errorf("checkov execution failed: %w", err)
	}
	
	// Parse Checkov JSON output
	return s.parseResults(output, scope)
}

// CheckovResult represents Checkov JSON output structure
type checkovResult struct {
	CheckType     string              `json:"check_type"`
	Results       checkovResultDetail `json:"results"`
	Summary       checkovSummary      `json:"summary"`
}

type checkovResultDetail struct {
	FailedChecks  []checkovCheck `json:"failed_checks"`
	PassedChecks  []checkovCheck `json:"passed_checks"`
	SkippedChecks []checkovCheck `json:"skipped_checks"`
}

type checkovCheck struct {
	CheckID       string   `json:"check_id"`
	CheckName     string   `json:"check_name"`
	CheckResult   map[string]interface{} `json:"check_result"`
	FilePath      string   `json:"file_path"`
	FileLineRange []int    `json:"file_line_range"`
	Resource      string   `json:"resource"`
	Guideline     string   `json:"guideline"`
}

type checkovSummary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// parseResults parses Checkov JSON output into findings
func (s *CheckovScanner) parseResults(output []byte, scope string) ([]findings.Finding, error) {
	var result checkovResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse checkov output: %w", err)
	}
	
	var results []findings.Finding
	
	// Process failed checks
	for _, check := range result.Results.FailedChecks {
		// Map to finding
		finding := findings.Finding{
			Title:       check.CheckName,
			Description: fmt.Sprintf("Checkov check %s failed for resource: %s", check.CheckID, check.Resource),
			Recommendation: check.Guideline,
			Files:       []string{check.FilePath},
			Severity:    mapCheckovSeverity(check.CheckID),
			Category:    findings.CategorySecurity,
		}
		
		// Generate ID based on scope
		prefix := findings.PrefixSecurity
		finding.ID = analysis.GenerateID(prefix, finding.Title, finding.Files)
		
		results = append(results, finding)
	}
	
	return results, nil
}

// mapCheckovSeverity maps Checkov check IDs to severity levels
// This is a simplified mapping - could be enhanced with a lookup table
func mapCheckovSeverity(checkID string) string {
	// High severity patterns (common critical security issues)
	if strings.Contains(checkID, "CKV_AWS_18") || // S3 bucket logging
		strings.Contains(checkID, "CKV_AWS_19") || // S3 bucket encryption
		strings.Contains(checkID, "CKV_AWS_20") || // S3 public access
		strings.Contains(checkID, "CKV_AWS_21") || // S3 versioning
		strings.Contains(checkID, "CKV_K8S_8") ||  // Privileged containers
		strings.Contains(checkID, "CKV_K8S_16") || // Container capabilities
		strings.Contains(checkID, "SECRETS") {     // Secrets detection
		return findings.SeverityHigh
	}
	
	// Medium severity (most other security checks)
	return findings.SeverityMedium
}
