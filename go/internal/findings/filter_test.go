package findings

import (
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/config"
)

func TestFilter(t *testing.T) {
	findings := []Finding{
		{
			Title:    "Security issue in production",
			Category: CategorySecurity,
			Severity: SeverityHigh,
			Files:    []string{"main.tf"},
		},
		{
			Title:    "Test security issue in sandbox",
			Category: CategorySecurity,
			Severity: SeverityLow,
			Files:    []string{"test/main.tf"},
		},
		{
			Title:    "Pipeline optimization",
			Category: CategoryPipeline,
			Severity: SeverityMedium,
			Files:    []string{".github/workflows/ci.yaml"},
		},
		{
			Title:    "Example security config",
			Category: CategorySecurity,
			Severity: SeverityLow,
			Files:    []string{"examples/config.tf"},
		},
	}

	cfg := &config.IgnoreConfig{
		Accepted: []config.AcceptedItem{
			{Title: "Security issue in production"},
		},
		IgnorePaths: []string{
			"examples/*",
		},
		IgnorePatterns: []string{
			"*sandbox*",
		},
	}

	filtered, ignoredCount := Filter(findings, cfg)

	// Should filter out: "Security issue in production" (accepted), "Test security issue in sandbox" (sandbox pattern), "Example security config" (examples path)
	if ignoredCount != 3 {
		t.Errorf("expected 3 ignored findings, got %d", ignoredCount)
	}

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered finding, got %d", len(filtered))
	}

	if len(filtered) > 0 && filtered[0].Title != "Pipeline optimization" {
		t.Errorf("expected Pipeline optimization to remain, got %s", filtered[0].Title)
	}
}

func TestShouldIgnore(t *testing.T) {
	cfg := &config.IgnoreConfig{
		Accepted: []config.AcceptedItem{
			{Title: "Some issue"},
		},
		IgnorePaths: []string{
			"examples/*",
		},
		IgnorePatterns: []string{
			"*test*",
			"demo*",
		},
	}
	acceptedTitles := cfg.GetAcceptedTitles()

	tests := []struct {
		name     string
		finding  Finding
		expected bool
	}{
		{
			name: "accepted title",
			finding: Finding{
				Title: "Some issue",
				Files: []string{"main.tf"},
			},
			expected: true,
		},
		{
			name: "matches ignore pattern",
			finding: Finding{
				Title: "Test security issue",
				Files: []string{"main.tf"},
			},
			expected: true,
		},
		{
			name: "matches ignore path",
			finding: Finding{
				Title: "Security config",
				Files: []string{"examples/config.tf"},
			},
			expected: true,
		},
		{
			name: "should not ignore",
			finding: Finding{
				Title: "Production security issue",
				Files: []string{"main.tf"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnore(tt.finding, cfg, acceptedTitles)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterBySeverity(t *testing.T) {
	findings := []Finding{
		{
			Title:    "Critical security issue",
			Severity: SeverityHigh,
		},
		{
			Title:    "Medium security issue",
			Severity: SeverityMedium,
		},
		{
			Title:    "Low priority optimization",
			Severity: SeverityLow,
		},
		{
			Title:    "Another high priority issue",
			Severity: SeverityHigh,
		},
	}

	tests := []struct {
		name             string
		minSeverity      string
		expectedCount    int
		expectedTitles   []string
	}{
		{
			name:          "low severity includes all",
			minSeverity:   SeverityLow,
			expectedCount: 4,
			expectedTitles:   []string{"Critical security issue", "Medium security issue", "Low priority optimization", "Another high priority issue"},
		},
		{
			name:          "medium severity filters out low",
			minSeverity:   SeverityMedium,
			expectedCount: 3,
			expectedTitles:   []string{"Critical security issue", "Medium security issue", "Another high priority issue"},
		},
		{
			name:          "high severity filters out medium and low",
			minSeverity:   SeverityHigh,
			expectedCount: 2,
			expectedTitles:   []string{"Critical security issue", "Another high priority issue"},
		},
		{
			name:          "empty min severity includes all",
			minSeverity:   "",
			expectedCount: 4,
			expectedTitles:   []string{"Critical security issue", "Medium security issue", "Low priority optimization", "Another high priority issue"},
		},
		{
			name:          "invalid severity includes all (safe fallback)",
			minSeverity:   "invalid",
			expectedCount: 4,
			expectedTitles:   []string{"Critical security issue", "Medium security issue", "Low priority optimization", "Another high priority issue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterBySeverity(findings, tt.minSeverity)
			
			if len(result) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(result))
			}
			
			// Check that the expected titles are present
			resultTitles := make(map[string]bool)
			for _, f := range result {
				resultTitles[f.Title] = true
			}
			
			for _, expectedTitle := range tt.expectedTitles {
				if !resultTitles[expectedTitle] {
					t.Errorf("expected finding %s to be present", expectedTitle)
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
