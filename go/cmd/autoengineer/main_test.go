package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

func TestSaveAndLoadFindings(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-findings.json")

	// Create test findings
	testFindings := []findings.Finding{
		{
			ID:             "SEC-001",
			Category:       "security",
			Title:          "Test Security Finding",
			Severity:       "high",
			Description:    "This is a test security finding",
			Recommendation: "Fix the security issue",
			Files:          []string{"test.tf"},
		},
		{
			ID:             "PIPE-001",
			Category:       "pipeline",
			Title:          "Test Pipeline Finding",
			Severity:       "medium",
			Description:    "This is a test pipeline finding",
			Recommendation: "Optimize the pipeline",
			Files:          []string{".github/workflows/test.yml"},
		},
	}

	// Test saving findings
	err := saveFindings(testFindings, testFile)
	if err != nil {
		t.Fatalf("saveFindings failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("findings file was not created")
	}

	// Test loading findings
	loadedFindings, err := loadFindings(testFile)
	if err != nil {
		t.Fatalf("loadFindings failed: %v", err)
	}

	// Verify loaded findings match original
	if len(loadedFindings) != len(testFindings) {
		t.Fatalf("expected %d findings, got %d", len(testFindings), len(loadedFindings))
	}

	for i, finding := range loadedFindings {
		if finding.ID != testFindings[i].ID {
			t.Errorf("finding %d: expected ID %s, got %s", i, testFindings[i].ID, finding.ID)
		}
		if finding.Title != testFindings[i].Title {
			t.Errorf("finding %d: expected Title %s, got %s", i, testFindings[i].Title, finding.Title)
		}
		if finding.Severity != testFindings[i].Severity {
			t.Errorf("finding %d: expected Severity %s, got %s", i, testFindings[i].Severity, finding.Severity)
		}
	}
}

func TestLoadFindings_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist.json")

	_, err := loadFindings(nonExistentFile)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	// Check if error message contains expected text
	expectedMsg := "findings file not found"
	if !strings.HasPrefix(err.Error(), expectedMsg) {
		t.Errorf("expected error message to start with '%s', got: %v", expectedMsg, err)
	}
}

func TestLoadFindings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON to file
	err := os.WriteFile(testFile, []byte("not valid json"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = loadFindings(testFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	// Check if error message contains expected text
	expectedMsg := "failed to parse findings file"
	if !strings.HasPrefix(err.Error(), expectedMsg) {
		t.Errorf("expected error message to start with '%s', got: %v", expectedMsg, err)
	}
}

func TestLoadFindings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.json")

	// Write empty JSON array to file
	err := os.WriteFile(testFile, []byte("[]"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loadedFindings, err := loadFindings(testFile)
	if err != nil {
		t.Fatalf("loadFindings failed for empty array: %v", err)
	}

	if len(loadedFindings) != 0 {
		t.Errorf("expected 0 findings for empty array, got %d", len(loadedFindings))
	}
}
