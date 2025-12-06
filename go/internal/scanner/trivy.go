package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/liam-witterick/autoengineer/go/internal/analysis"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// TrivyScanner implements Scanner for Trivy
type TrivyScanner struct {
	binaryPath string
}

const (
	trivySeverities = "CRITICAL,HIGH,MEDIUM,LOW"
)

// NewTrivyScanner creates a new Trivy scanner
func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{
		binaryPath: "trivy",
	}
}

// Name returns the scanner name
func (s *TrivyScanner) Name() string {
	return "trivy"
}

// Type returns the scanner type
func (s *TrivyScanner) Type() ScannerType {
	return TypeLocal
}

// IsInstalled checks if Trivy is available
func (s *TrivyScanner) IsInstalled() bool {
	_, err := exec.LookPath(s.binaryPath)
	return err == nil
}

// Version returns the Trivy version
func (s *TrivyScanner) Version() string {
	if !s.IsInstalled() {
		return ""
	}
	
	cmd := exec.Command(s.binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "installed"
	}
	
	// Parse version from output (first line typically)
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	
	return "installed"
}

// Run executes Trivy and returns findings
func (s *TrivyScanner) Run(ctx context.Context, scope string) ([]findings.Finding, error) {
	// Run trivy with JSON output for config scanning
	args := []string{
		"config",
		"--format", "json",
		"--exit-code", "0", // Don't fail on findings
		"--quiet",
		".",
	}
	
	// Add severity filter based on scope
	if scope == "security" || scope == "all" {
		args = append(args, "--severity", trivySeverities)
	}
	
	cmd := exec.CommandContext(ctx, s.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if there's any output despite error
		if len(output) == 0 {
			return nil, fmt.Errorf("trivy execution failed: %w", err)
		}
		// Continue to parse even if there was an error
	}
	
	// Parse Trivy JSON output
	return s.parseResults(output, scope)
}

// TrivyResult represents Trivy JSON output structure
type trivyResult struct {
	Results []trivyFileResult `json:"Results"`
}

type trivyFileResult struct {
	Target         string              `json:"Target"`
	Class          string              `json:"Class"`
	Type           string              `json:"Type"`
	Misconfigurations []trivyMisconfig `json:"Misconfigurations"`
}

type trivyMisconfig struct {
	Type       string `json:"Type"`
	ID         string `json:"ID"`
	Title      string `json:"Title"`
	Description string `json:"Description"`
	Message    string `json:"Message"`
	Resolution string `json:"Resolution"`
	Severity   string `json:"Severity"`
	PrimaryURL string `json:"PrimaryURL"`
	References []string `json:"References"`
}

// parseResults parses Trivy JSON output into findings
func (s *TrivyScanner) parseResults(output []byte, scope string) ([]findings.Finding, error) {
	var result trivyResult
	if err := json.Unmarshal(output, &result); err != nil {
		// If JSON parsing fails, return empty results (Trivy may not have found anything)
		return []findings.Finding{}, nil
	}
	
	var results []findings.Finding
	
	// Process all misconfigurations
	for _, fileResult := range result.Results {
		for _, misconfig := range fileResult.Misconfigurations {
			// Map to finding
			description := misconfig.Description
			if misconfig.Message != "" {
				description = misconfig.Message
			}
			
			finding := findings.Finding{
				Title:       misconfig.Title,
				Description: description,
				Recommendation: misconfig.Resolution,
				Files:       []string{fileResult.Target},
				Severity:    mapTrivySeverity(misconfig.Severity),
				Category:    findings.CategorySecurity,
			}
			
			// Generate ID
			prefix := findings.PrefixSecurity
			finding.ID = analysis.GenerateID(prefix, finding.Title, finding.Files)
			
			results = append(results, finding)
		}
	}
	
	return results, nil
}

// mapTrivySeverity maps Trivy severity levels to our severity levels
func mapTrivySeverity(severity string) string {
	switch strings.ToUpper(severity) {
	case "CRITICAL", "HIGH":
		return findings.SeverityHigh
	case "MEDIUM":
		return findings.SeverityMedium
	case "LOW", "UNKNOWN":
		return findings.SeverityLow
	default:
		return findings.SeverityMedium
	}
}
