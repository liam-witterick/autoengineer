package findings

import (
	"strings"

	"github.com/liam-witterick/infra-review/go/internal/config"
)

// Filter filters findings based on ignore configuration
func Filter(findings []Finding, cfg *config.IgnoreConfig) (filtered []Finding, ignoredCount int) {
	acceptedIDs := cfg.GetAcceptedIDs()
	filtered = make([]Finding, 0, len(findings))

	for _, finding := range findings {
		if shouldIgnore(finding, cfg, acceptedIDs) {
			ignoredCount++
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered, ignoredCount
}

// FilterBySeverity filters findings by minimum severity level
// Returns only findings at or above the specified minimum severity
func FilterBySeverity(findings []Finding, minSeverity string) []Finding {
	if minSeverity == "" || minSeverity == SeverityLow {
		// No filtering needed - low includes all severities
		return findings
	}

	// Validate severity before using it
	if !ValidateSeverity(minSeverity) {
		// Invalid severity - return all findings to be safe
		return findings
	}

	severityOrder := map[string]int{
		SeverityHigh:   3,
		SeverityMedium: 2,
		SeverityLow:    1,
	}

	minLevel := severityOrder[minSeverity]
	filtered := make([]Finding, 0, len(findings))

	for _, finding := range findings {
		if severityOrder[finding.Severity] >= minLevel {
			filtered = append(filtered, finding)
		}
	}

	return filtered
}

// ValidateSeverity checks if a severity string is valid
func ValidateSeverity(severity string) bool {
	return severity == SeverityHigh || severity == SeverityMedium || severity == SeverityLow
}

// shouldIgnore determines if a finding should be ignored based on config
func shouldIgnore(finding Finding, cfg *config.IgnoreConfig, acceptedIDs map[string]bool) bool {
	// Check if ID is in accepted list
	if acceptedIDs[finding.ID] {
		return true
	}

	// Check if title matches any ignore patterns
	titleLower := strings.ToLower(finding.Title)
	for _, pattern := range cfg.IgnorePatterns {
		patternLower := strings.ToLower(pattern)
		// Convert glob pattern to simple substring match
		// * at start or end becomes prefix/suffix match
		if strings.HasPrefix(patternLower, "*") && strings.HasSuffix(patternLower, "*") {
			// Pattern like *sandbox*
			if strings.Contains(titleLower, strings.Trim(patternLower, "*")) {
				return true
			}
		} else if strings.HasPrefix(patternLower, "*") {
			// Pattern like *test
			if strings.HasSuffix(titleLower, strings.TrimPrefix(patternLower, "*")) {
				return true
			}
		} else if strings.HasSuffix(patternLower, "*") {
			// Pattern like test*
			if strings.HasPrefix(titleLower, strings.TrimSuffix(patternLower, "*")) {
				return true
			}
		} else {
			// Exact substring match
			if strings.Contains(titleLower, patternLower) {
				return true
			}
		}
	}

	// Check if any file matches ignore paths
	for _, file := range finding.Files {
		if cfg.MatchesPath(file) {
			return true
		}
	}

	return false
}
