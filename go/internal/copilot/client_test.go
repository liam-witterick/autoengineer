package copilot

import (
	"strings"
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
	"github.com/liam-witterick/autoengineer/go/internal/issues"
)

func TestBuildDeduplicationPrompt(t *testing.T) {
	tests := []struct {
		name            string
		newFindings     []findings.Finding
		existingIssues  []issues.SearchResult
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "with findings and existing issues",
			newFindings: []findings.Finding{
				{
					ID:          "SEC-001",
					Title:       "S3 bucket lacks encryption",
					Category:    findings.CategorySecurity,
					Severity:    findings.SeverityHigh,
					Description: "Encryption not enabled",
					Files:       []string{"storage.tf"},
				},
				{
					ID:          "INFRA-002",
					Title:       "MSK uses public subnets",
					Category:    findings.CategoryInfra,
					Severity:    findings.SeverityMedium,
					Description: "MSK should use private subnets",
					Files:       []string{"kafka.tf"},
				},
			},
			existingIssues: []issues.SearchResult{
				{
					Number: 43,
					Title:  "🔴 Security: Restrict MSK Security Group CIDR Blocks",
				},
			},
			wantContains: []string{
				"EXISTING TRACKED ISSUES",
				"Issue #43",
				"Restrict MSK Security Group CIDR Blocks",
				"NEW FINDINGS TO DEDUPLICATE",
				"SEC-001",
				"S3 bucket lacks encryption",
				"INFRA-002",
				"MSK uses public subnets",
				"INSTRUCTIONS",
				"Merge findings that describe the same underlying issue",
				"even if they're from different categories",
				"Remove any findings that are duplicates",
			},
			wantNotContains: []string{},
		},
		{
			name: "with findings only, no existing issues",
			newFindings: []findings.Finding{
				{
					ID:       "PIPE-001",
					Title:    "CI/CD pipeline needs optimization",
					Category: findings.CategoryPipeline,
					Severity: findings.SeverityLow,
				},
			},
			existingIssues: []issues.SearchResult{},
			wantContains: []string{
				"NEW FINDINGS TO DEDUPLICATE",
				"PIPE-001",
				"CI/CD pipeline needs optimization",
				"INSTRUCTIONS",
			},
			wantNotContains: []string{
				"EXISTING TRACKED ISSUES",
			},
		},
		{
			name:           "empty findings",
			newFindings:    []findings.Finding{},
			existingIssues: []issues.SearchResult{},
			wantContains: []string{
				"NEW FINDINGS TO DEDUPLICATE",
				"INSTRUCTIONS",
			},
			wantNotContains: []string{
				"EXISTING TRACKED ISSUES",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildDeduplicationPrompt(tt.newFindings, tt.existingIssues)

			for _, want := range tt.wantContains {
				if !contains(prompt, want) {
					t.Errorf("prompt should contain %q", want)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if contains(prompt, notWant) {
					t.Errorf("prompt should NOT contain %q", notWant)
				}
			}
		})
	}
}

func TestBuildDeduplicationPromptFormat(t *testing.T) {
	newFindings := []findings.Finding{
		{
			ID:             "SEC-001",
			Title:          "Security issue",
			Category:       findings.CategorySecurity,
			Severity:       findings.SeverityHigh,
			Description:    "Test description",
			Recommendation: "Test recommendation",
			Files:          []string{"file1.tf", "file2.tf"},
		},
	}

	prompt := buildDeduplicationPrompt(newFindings, []issues.SearchResult{})

	// Check that prompt contains JSON structure
	if !contains(prompt, `"id"`) {
		t.Error("prompt should contain JSON id field")
	}
	if !contains(prompt, `"category"`) {
		t.Error("prompt should contain JSON category field")
	}
	if !contains(prompt, `"title"`) {
		t.Error("prompt should contain JSON title field")
	}
	if !contains(prompt, `"severity"`) {
		t.Error("prompt should contain JSON severity field")
	}
	if !contains(prompt, `"files"`) {
		t.Error("prompt should contain JSON files field")
	}
}

func TestBuildDeduplicationPromptMultipleIssues(t *testing.T) {
	existingIssues := []issues.SearchResult{
		{Number: 1, Title: "Issue 1"},
		{Number: 2, Title: "Issue 2"},
		{Number: 3, Title: "Issue 3"},
	}

	prompt := buildDeduplicationPrompt([]findings.Finding{}, existingIssues)

	// Check all issues are mentioned
	if !contains(prompt, "Issue #1") {
		t.Error("prompt should contain Issue #1")
	}
	if !contains(prompt, "Issue #2") {
		t.Error("prompt should contain Issue #2")
	}
	if !contains(prompt, "Issue #3") {
		t.Error("prompt should contain Issue #3")
	}
	if !contains(prompt, "Issue 1") {
		t.Error("prompt should contain Issue 1 title")
	}
	if !contains(prompt, "Issue 2") {
		t.Error("prompt should contain Issue 2 title")
	}
	if !contains(prompt, "Issue 3") {
		t.Error("prompt should contain Issue 3 title")
	}
}

func TestBuildDeduplicationPromptInstructions(t *testing.T) {
	prompt := buildDeduplicationPrompt([]findings.Finding{}, []issues.SearchResult{})

	requiredInstructions := []string{
		"Merge findings that describe the same underlying issue",
		"even if they're from different categories",
		"keep the finding with the highest severity",
		"combine the file lists",
		"Remove any findings that are duplicates",
		"Return ONLY the deduplicated findings as a JSON array",
	}

	for _, instruction := range requiredInstructions {
		if !contains(prompt, instruction) {
			t.Errorf("prompt should contain instruction: %q", instruction)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestExtractJSON tests the extractJSON function with various input formats
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "markdown code block",
			input: "Here is the result:\n```json\n[{\"id\": \"1\", \"name\": \"test\"}]\n```\nDone!",
			expected: "[{\"id\": \"1\", \"name\": \"test\"}]",
		},
		{
			name: "inline JSON array starting with [{",
			input: "The output is: [{\"id\": \"1\", \"name\": \"test\"}] and that's it.",
			expected: "[{\"id\": \"1\", \"name\": \"test\"}]",
		},
		{
			name: "JSON array with nested objects",
			input: "Result: [{\"id\": \"1\", \"data\": {\"nested\": \"value\"}}, {\"id\": \"2\"}]",
			expected: "[{\"id\": \"1\", \"data\": {\"nested\": \"value\"}}, {\"id\": \"2\"}]",
		},
		{
			name: "JSON array with ellipsis after",
			input: "[{\"id\": \"1\"}]... and more text",
			expected: "[{\"id\": \"1\"}]",
		},
		{
			name: "line-by-line JSON array",
			input: "[\n  {\"id\": \"1\"},\n  {\"id\": \"2\"}\n]",
			expected: "[\n  {\"id\": \"1\"},\n  {\"id\": \"2\"}\n]",
		},
		{
			name: "markdown code block with invalid JSON - returns empty",
			input: "```json\n[{\"id\": 1, \"name\": test}]\n```",
			expected: "",
		},
		{
			name: "inline invalid JSON - returns empty",
			input: "The output is: [{\"id\": 1, \"name\": test.}] invalid",
			expected: "",
		},
		{
			name: "no JSON present",
			input: "This is just plain text without any JSON",
			expected: "",
		},
		{
			name: "empty string",
			input: "",
			expected: "",
		},
		{
			name: "JSON array with nested arrays",
			input: "[{\"id\": \"1\", \"items\": [\"a\", \"b\"]}, {\"id\": \"2\", \"items\": []}]",
			expected: "[{\"id\": \"1\", \"items\": [\"a\", \"b\"]}, {\"id\": \"2\", \"items\": []}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractJSONPriority(t *testing.T) {
	// Test that markdown code blocks are prioritized over inline JSON
	input := "Some text [{\"inline\": \"json\"}] and ```json\n[{\"code\": \"block\"}]\n```"
	result := extractJSON(input)
	expected := "[{\"code\": \"block\"}]"
	
	if result != expected {
		t.Errorf("extractJSON() should prioritize code blocks, got %q, want %q", result, expected)
	}
}

func TestExtractJSONValidation(t *testing.T) {
	// Test that invalid JSON is rejected even if pattern matches
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "invalid JSON in code block",
			input: "```json\n[{id: 1}]\n```",
		},
		{
			name: "malformed inline JSON",
			input: "[{\"id\": 1, \"name\": }]",
		},
		{
			name: "incomplete JSON array",
			input: "[{\"id\": 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != "" {
				t.Errorf("extractJSON() should return empty string for invalid JSON, got %q", result)
			}
		})
	}
}
