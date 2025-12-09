package findings

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{"high", SeverityHigh, "🔴"},
		{"medium", SeverityMedium, "🟡"},
		{"low", SeverityLow, "🟢"},
		{"unknown", "unknown", "⚪"},
		{"empty", "", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityEmoji(tt.severity)
			if got != tt.want {
				t.Errorf("SeverityEmoji(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestDisplaySummary(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityHigh, Category: CategorySecurity},
		{Severity: SeverityHigh, Category: CategorySecurity},
		{Severity: SeverityMedium, Category: CategoryPipeline},
		{Severity: SeverityLow, Category: CategoryInfra},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplaySummary(findings)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected elements
	expectedStrings := []string{
		"Summary:",
		"High: 2",
		"Medium: 1",
		"Low: 1",
		"Total: 4",
		"Security:",
		"2 finding(s)",
		"Pipeline:",
		"1 finding(s)",
		"Infrastructure:",
		"1 finding(s)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("DisplaySummary() output missing expected string: %q", expected)
		}
	}
}

func TestDisplayFindings(t *testing.T) {
	findings := []Finding{
		{
			Title:       "Test Security Finding",
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
			Files:       []string{"main.go", "auth.go"},
			Description: "This is a test security issue",
		},
		{
			Title:       "Test Pipeline Finding",
			Severity:    SeverityMedium,
			Category:    CategoryPipeline,
			Files:       []string{".github/workflows/ci.yml"},
			Description: "This is a test pipeline issue",
		},
	}

	// Test default options
	t.Run("default options", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		DisplayFindings(findings, DefaultDisplayOptions())

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Should show findings
		if !strings.Contains(output, "Test Security Finding") {
			t.Error("DisplayFindings() should show finding titles")
		}
		// Should not show category by default
		if strings.Contains(output, "Category:") {
			t.Error("DisplayFindings() should not show category with default options")
		}
		// Should not show description by default
		if strings.Contains(output, "Description:") {
			t.Error("DisplayFindings() should not show description with default options")
		}
	})

	// Test detailed options
	t.Run("detailed options", func(t *testing.T) {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		DisplayFindings(findings, DetailedDisplayOptions())

		w.Close()
		os.Stdout = old

		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Should show all details
		if !strings.Contains(output, "Category:") {
			t.Error("DisplayFindings() should show category with detailed options")
		}
		if !strings.Contains(output, "Description:") {
			t.Error("DisplayFindings() should show description with detailed options")
		}
		if !strings.Contains(output, "security") {
			t.Error("DisplayFindings() should show category value")
		}
	})
}

func TestDisplayOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := DefaultDisplayOptions()
		if opts.ShowCategory {
			t.Error("DefaultDisplayOptions() ShowCategory should be false")
		}
		if opts.ShowDescription {
			t.Error("DefaultDisplayOptions() ShowDescription should be false")
		}
		if opts.ShowAll {
			t.Error("DefaultDisplayOptions() ShowAll should be false")
		}
		if opts.MaxDisplay != 5 {
			t.Errorf("DefaultDisplayOptions() MaxDisplay = %d, want 5", opts.MaxDisplay)
		}
	})

	t.Run("detailed options", func(t *testing.T) {
		opts := DetailedDisplayOptions()
		if !opts.ShowCategory {
			t.Error("DetailedDisplayOptions() ShowCategory should be true")
		}
		if !opts.ShowDescription {
			t.Error("DetailedDisplayOptions() ShowDescription should be true")
		}
		if !opts.ShowAll {
			t.Error("DetailedDisplayOptions() ShowAll should be true")
		}
		if opts.TruncateDesc != 150 {
			t.Errorf("DetailedDisplayOptions() TruncateDesc = %d, want 150", opts.TruncateDesc)
		}
	})
}
