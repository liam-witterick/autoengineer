package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInstructions(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	t.Run("file does not exist", func(t *testing.T) {
		instructions, err := LoadInstructions("nonexistent.md")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if instructions != "" {
			t.Errorf("expected empty string, got %q", instructions)
		}
	})

	t.Run("file exists with content", func(t *testing.T) {
		content := "Focus on security issues\nIgnore test files"
		testFile := filepath.Join(tmpDir, "test-instructions.md")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		instructions, err := LoadInstructions(testFile)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !strings.Contains(instructions, "CUSTOM INSTRUCTIONS FROM USER:") {
			t.Error("expected formatted instructions to contain header")
		}
		if !strings.Contains(instructions, content) {
			t.Errorf("expected instructions to contain original content")
		}
		if !strings.Contains(instructions, "END CUSTOM INSTRUCTIONS") {
			t.Error("expected formatted instructions to contain footer")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "empty-instructions.md")
		if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		instructions, err := LoadInstructions(testFile)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if instructions != "" {
			t.Errorf("expected empty string for empty file, got %q", instructions)
		}
	})

	t.Run("file with permission denied", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "noperm-instructions.md")
		if err := os.WriteFile(testFile, []byte("test"), 0000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(testFile, 0644) // Cleanup

		instructions, err := LoadInstructions(testFile)
		if err == nil {
			t.Error("expected error for permission denied, got nil")
		}
		if instructions != "" {
			t.Errorf("expected empty string on error, got %q", instructions)
		}
	})
}

func TestLoadDefaultInstructions(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	githubDir := filepath.Join(tmpDir, ".github")
	if err := os.Mkdir(githubDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	t.Run("default file does not exist", func(t *testing.T) {
		instructions, err := LoadDefaultInstructions()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if instructions != "" {
			t.Errorf("expected empty string, got %q", instructions)
		}
	})

	t.Run("default file exists", func(t *testing.T) {
		content := "## High Priority\n- Security issues\n- Performance issues"
		defaultFile := filepath.Join(githubDir, "copilot-instructions.md")
		if err := os.WriteFile(defaultFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		instructions, err := LoadDefaultInstructions()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if !strings.Contains(instructions, content) {
			t.Error("expected instructions to contain original content")
		}
		if !strings.Contains(instructions, "CUSTOM INSTRUCTIONS FROM USER:") {
			t.Error("expected formatted instructions")
		}
	})
}

func TestCheckInstructionsExists(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	githubDir := filepath.Join(tmpDir, ".github")
	if err := os.Mkdir(githubDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	t.Run("file does not exist", func(t *testing.T) {
		exists := CheckInstructionsExists()
		if exists {
			t.Error("expected false, got true")
		}
	})

	t.Run("file exists", func(t *testing.T) {
		defaultFile := filepath.Join(githubDir, "copilot-instructions.md")
		if err := os.WriteFile(defaultFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		exists := CheckInstructionsExists()
		if !exists {
			t.Error("expected true, got false")
		}
	})
}

func TestInstructionsFormatting(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	content := "Test instruction content"
	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	instructions, err := LoadInstructions(testFile)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the format is correct
	expectedStart := "\n\nCUSTOM INSTRUCTIONS FROM USER:\n"
	expectedEnd := "\nEND CUSTOM INSTRUCTIONS\n"
	
	if !strings.HasPrefix(instructions, expectedStart) {
		t.Errorf("expected instructions to start with %q", expectedStart)
	}
	if !strings.HasSuffix(instructions, expectedEnd) {
		t.Errorf("expected instructions to end with %q", expectedEnd)
	}
	if !strings.Contains(instructions, content) {
		t.Error("expected instructions to contain original content")
	}
}

func TestFormatInstructions(t *testing.T) {
	t.Run("with text", func(t *testing.T) {
		text := "Test instructions"
		formatted := FormatInstructions(text)
		
		if !strings.Contains(formatted, "CUSTOM INSTRUCTIONS FROM USER:") {
			t.Error("expected formatted instructions to contain header")
		}
		if !strings.Contains(formatted, text) {
			t.Error("expected formatted instructions to contain original text")
		}
		if !strings.Contains(formatted, "END CUSTOM INSTRUCTIONS") {
			t.Error("expected formatted instructions to contain footer")
		}
	})
	
	t.Run("with empty text", func(t *testing.T) {
		formatted := FormatInstructions("")
		if formatted != "" {
			t.Errorf("expected empty string, got %q", formatted)
		}
	})
}
