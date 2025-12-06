package findings

import (
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/config"
)

func TestFilter(t *testing.T) {
	findings := []Finding{
		{
			ID:       "SEC-001",
			Title:    "Security issue in production",
			Category: CategorySecurity,
			Severity: SeverityHigh,
			Files:    []string{"main.tf"},
		},
		{
			ID:       "SEC-002",
			Title:    "Test security issue in sandbox",
			Category: CategorySecurity,
			Severity: SeverityLow,
			Files:    []string{"test/main.tf"},
		},
		{
			ID:       "PIPE-001",
			Title:    "Pipeline optimization",
			Category: CategoryPipeline,
			Severity: SeverityMedium,
			Files:    []string{".github/workflows/ci.yaml"},
		},
		{
			ID:       "SEC-003",
			Title:    "Example security config",
			Category: CategorySecurity,
			Severity: SeverityLow,
			Files:    []string{"examples/config.tf"},
		},
	}

	cfg := &config.IgnoreConfig{
		Accepted: []config.AcceptedItem{
			{ID: "SEC-001"},
		},
		IgnorePaths: []string{
			"examples/*",
		},
		IgnorePatterns: []string{
			"*sandbox*",
		},
	}

	filtered, ignoredCount := Filter(findings, cfg)

	// Should filter out: SEC-001 (accepted), SEC-002 (sandbox pattern), SEC-003 (examples path)
	if ignoredCount != 3 {
		t.Errorf("expected 3 ignored findings, got %d", ignoredCount)
	}

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered finding, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].ID != "PIPE-001" {
		t.Errorf("expected PIPE-001 to remain, got %s", filtered[0].ID)
	}
}

func TestShouldIgnore(t *testing.T) {
	cfg := &config.IgnoreConfig{
		Accepted: []config.AcceptedItem{
			{ID: "SEC-001"},
		},
		IgnorePaths: []string{
			"examples/*",
		},
		IgnorePatterns: []string{
			"*test*",
			"demo*",
		},
	}
	acceptedIDs := cfg.GetAcceptedIDs()

	tests := []struct {
		name     string
		finding  Finding
		expected bool
	}{
		{
			name: "accepted ID",
			finding: Finding{
				ID:    "SEC-001",
				Title: "Some issue",
				Files: []string{"main.tf"},
			},
			expected: true,
		},
		{
			name: "matches ignore pattern",
			finding: Finding{
				ID:    "SEC-002",
				Title: "Test security issue",
				Files: []string{"main.tf"},
			},
			expected: true,
		},
		{
			name: "matches ignore path",
			finding: Finding{
				ID:    "SEC-003",
				Title: "Security config",
				Files: []string{"examples/config.tf"},
			},
			expected: true,
		},
		{
			name: "should not ignore",
			finding: Finding{
				ID:    "SEC-004",
				Title: "Production security issue",
				Files: []string{"main.tf"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnore(tt.finding, cfg, acceptedIDs)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterBySeverity(t *testing.T) {
	findings := []Finding{
		{
			ID:       "SEC-001",
			Title:    "Critical security issue",
			Severity: SeverityHigh,
		},
		{
			ID:       "SEC-002",
			Title:    "Medium security issue",
			Severity: SeverityMedium,
		},
		{
			ID:       "PIPE-001",
			Title:    "Low priority optimization",
			Severity: SeverityLow,
		},
		{
			ID:       "INFRA-001",
			Title:    "Another high priority issue",
			Severity: SeverityHigh,
		},
	}

	tests := []struct {
		name             string
		minSeverity      string
		expectedCount    int
		expectedIDs      []string
	}{
		{
			name:          "low severity includes all",
			minSeverity:   SeverityLow,
			expectedCount: 4,
			expectedIDs:   []string{"SEC-001", "SEC-002", "PIPE-001", "INFRA-001"},
		},
		{
			name:          "medium severity filters out low",
			minSeverity:   SeverityMedium,
			expectedCount: 3,
			expectedIDs:   []string{"SEC-001", "SEC-002", "INFRA-001"},
		},
		{
			name:          "high severity filters out medium and low",
			minSeverity:   SeverityHigh,
			expectedCount: 2,
			expectedIDs:   []string{"SEC-001", "INFRA-001"},
		},
		{
			name:          "empty min severity includes all",
			minSeverity:   "",
			expectedCount: 4,
			expectedIDs:   []string{"SEC-001", "SEC-002", "PIPE-001", "INFRA-001"},
		},
		{
			name:          "invalid severity includes all (safe fallback)",
			minSeverity:   "invalid",
			expectedCount: 4,
			expectedIDs:   []string{"SEC-001", "SEC-002", "PIPE-001", "INFRA-001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterBySeverity(findings, tt.minSeverity)
			
			if len(result) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(result))
			}
			
			// Check that the expected IDs are present
			resultIDs := make(map[string]bool)
			for _, f := range result {
				resultIDs[f.ID] = true
			}
			
			for _, expectedID := range tt.expectedIDs {
				if !resultIDs[expectedID] {
					t.Errorf("expected finding %s to be present", expectedID)
				}
			}
		})
	}
}

func TestValidateSeverity(t *testing.T) {
	tests := []struct {
		severity string
		expected bool
	}{
		{SeverityHigh, true},
		{SeverityMedium, true},
		{SeverityLow, true},
		{"invalid", false},
		{"", false},
		{"HIGH", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := ValidateSeverity(tt.severity)
			if result != tt.expected {
				t.Errorf("ValidateSeverity(%q) = %v, want %v", tt.severity, result, tt.expected)
			}
		})
	}
}
