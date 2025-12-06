package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreConfig(t *testing.T) {
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

	t.Run("no config file", func(t *testing.T) {
		cfg, err := LoadIgnoreConfig()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if cfg == nil {
			t.Error("expected non-nil config")
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		configContent := `
accepted:
  - id: "SEC-abc123"
    reason: "Legacy system"
    accepted_by: "security-team"
    accepted_date: "2025-01-15"

ignore_paths:
  - "examples/*"
  - "**/testdata/**"

ignore_patterns:
  - "*sandbox*"
  - "*test*"

disabled_scopes:
  - pipeline
`
		configPath := filepath.Join(githubDir, "autoengineer-ignore.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadIgnoreConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(cfg.Accepted) != 1 {
			t.Errorf("expected 1 accepted item, got %d", len(cfg.Accepted))
		}
		if cfg.Accepted[0].ID != "SEC-abc123" {
			t.Errorf("expected ID SEC-abc123, got %s", cfg.Accepted[0].ID)
		}

		if len(cfg.IgnorePaths) != 2 {
			t.Errorf("expected 2 ignore paths, got %d", len(cfg.IgnorePaths))
		}

		if len(cfg.IgnorePatterns) != 2 {
			t.Errorf("expected 2 ignore patterns, got %d", len(cfg.IgnorePatterns))
		}

		if len(cfg.DisabledScopes) != 1 {
			t.Errorf("expected 1 disabled scope, got %d", len(cfg.DisabledScopes))
		}
	})
}

func TestGetAcceptedIDs(t *testing.T) {
	cfg := &IgnoreConfig{
		Accepted: []AcceptedItem{
			{ID: "SEC-123"},
			{ID: "PIPE-456"},
		},
	}

	ids := cfg.GetAcceptedIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
	if !ids["SEC-123"] {
		t.Error("expected SEC-123 to be in accepted IDs")
	}
	if !ids["PIPE-456"] {
		t.Error("expected PIPE-456 to be in accepted IDs")
	}
}

func TestIsScopeDisabled(t *testing.T) {
	cfg := &IgnoreConfig{
		DisabledScopes: []string{"pipeline", "infra"},
	}

	tests := []struct {
		scope    string
		expected bool
	}{
		{"pipeline", true},
		{"infra", true},
		{"security", false},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			result := cfg.IsScopeDisabled(tt.scope)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMatchesPath(t *testing.T) {
	cfg := &IgnoreConfig{
		IgnorePaths: []string{
			"examples/*",
			"test/fixtures/*",
			"*.example.*",
		},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"examples/demo.tf", true},
		{"test/fixtures/config.yaml", true},
		{"main.example.tf", true},
		{"src/main.tf", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := cfg.MatchesPath(tt.path)
			if result != tt.expected {
				t.Errorf("path %s: expected %v, got %v", tt.path, tt.expected, result)
			}
		})
	}
}
