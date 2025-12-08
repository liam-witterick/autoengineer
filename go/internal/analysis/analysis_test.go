package analysis

import (
	"strings"
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/issues"
)

func TestBuildExistingContext_Empty(t *testing.T) {
	result := BuildExistingContext([]issues.SearchResult{})
	if result != "" {
		t.Errorf("Expected empty string for empty issues, got: %s", result)
	}
}

func TestBuildExistingContext_SingleIssue(t *testing.T) {
	issues := []issues.SearchResult{
		{Number: 123, Title: "🔴 Security vulnerability in auth module"},
	}
	
	result := BuildExistingContext(issues)
	
	if !strings.Contains(result, "SKIP these already-tracked issues") {
		t.Error("Expected context to contain 'SKIP these already-tracked issues'")
	}
	
	if !strings.Contains(result, "🔴 Security vulnerability in auth module") {
		t.Error("Expected context to contain issue title")
	}
	
	if !strings.Contains(result, "Issue #123") {
		t.Error("Expected context to contain issue number")
	}
	
	if !strings.Contains(result, "Do NOT report findings") {
		t.Error("Expected context to contain instruction not to report")
	}
}

func TestBuildExistingContext_MultipleIssues(t *testing.T) {
	issues := []issues.SearchResult{
		{Number: 123, Title: "🔴 Security vulnerability in auth module"},
		{Number: 456, Title: "🟡 Missing cache strategy in build workflow"},
	}
	
	result := BuildExistingContext(issues)
	
	if !strings.Contains(result, "Issue #123") {
		t.Error("Expected context to contain first issue number")
	}
	
	if !strings.Contains(result, "Issue #456") {
		t.Error("Expected context to contain second issue number")
	}
	
	if !strings.Contains(result, "Security vulnerability") {
		t.Error("Expected context to contain first issue title")
	}
	
	if !strings.Contains(result, "Missing cache strategy") {
		t.Error("Expected context to contain second issue title")
	}
}
