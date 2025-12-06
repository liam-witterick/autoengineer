package scanner

import (
	"context"
	"sync"

	"github.com/liam-witterick/autoengineer/go/internal/config"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// Manager manages scanner execution and coordination
type Manager struct {
	scanners []Scanner
	config   *config.ScannerConfig
}

// NewManager creates a new scanner manager
func NewManager(cfg *config.ScannerConfig) *Manager {
	// Initialize default scanners (Checkov and Trivy)
	defaultScanners := []Scanner{
		NewCheckovScanner(),
		NewTrivyScanner(),
	}
	
	return &Manager{
		scanners: defaultScanners,
		config:   cfg,
	}
}

// DetectScanners returns status of all scanners
func (m *Manager) DetectScanners() []ScannerStatus {
	var statuses []ScannerStatus
	
	for _, scanner := range m.scanners {
		installed := scanner.IsInstalled()
		enabled := m.isScannerEnabled(scanner.Name(), installed)
		
		status := ScannerStatus{
			Name:      scanner.Name(),
			Type:      scanner.Type(),
			Installed: installed,
			Version:   "",
			Enabled:   enabled,
			Skipped:   !enabled,
		}
		
		if installed {
			status.Version = scanner.Version()
		}
		
		if !enabled {
			if !installed {
				status.Reason = "not installed"
			} else if m.config != nil && m.config.IsDisabled(scanner.Name()) {
				status.Reason = "disabled in config"
			}
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses
}

// isScannerEnabled checks if a scanner should be run
func (m *Manager) isScannerEnabled(name string, installed bool) bool {
	// If config explicitly disables it, don't run
	if m.config != nil && m.config.IsDisabled(name) {
		return false
	}
	
	// For local scanners (checkov, trivy), only run if installed
	// For cloud scanners, check if explicitly enabled in config
	switch name {
	case "checkov", "trivy":
		return installed
	default:
		// Cloud scanners need explicit enablement
		if m.config != nil {
			return m.config.IsEnabled(name)
		}
		return false
	}
}

// RunAll runs all enabled scanners in parallel
func (m *Manager) RunAll(ctx context.Context, scope string) ([]findings.Finding, []ScannerStatus) {
	statuses := m.DetectScanners()
	
	// Filter to enabled scanners
	var enabledScanners []Scanner
	for i, scanner := range m.scanners {
		if statuses[i].Enabled {
			enabledScanners = append(enabledScanners, scanner)
		}
	}
	
	if len(enabledScanners) == 0 {
		return []findings.Finding{}, statuses
	}
	
	// Run scanners in parallel
	results := make(chan ScanResult, len(enabledScanners))
	var wg sync.WaitGroup
	
	for _, scanner := range enabledScanners {
		wg.Add(1)
		go func(s Scanner) {
			defer wg.Done()
			
			scanFindings, err := s.Run(ctx, scope)
			results <- ScanResult{
				Scanner:  s.Name(),
				Findings: scanFindings,
				Error:    err,
			}
		}(scanner)
	}
	
	// Wait for all to complete
	wg.Wait()
	close(results)
	
	// Collect all findings and update statuses
	var allFindings []findings.Finding
	scanResults := make(map[string]ScanResult)
	
	for result := range results {
		scanResults[result.Scanner] = result
		if result.Error == nil {
			allFindings = append(allFindings, result.Findings...)
		}
	}
	
	// Update statuses with results
	for i := range statuses {
		if statuses[i].Enabled {
			if result, ok := scanResults[statuses[i].Name]; ok {
				statuses[i].Ran = result.Error == nil
				statuses[i].Found = len(result.Findings)
				statuses[i].Error = result.Error
			}
		}
	}
	
	return allFindings, statuses
}
