package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IgnoreConfig represents the autoengineer-ignore.yaml configuration
type IgnoreConfig struct {
	Accepted       []AcceptedItem `yaml:"accepted"`
	IgnorePaths    []string       `yaml:"ignore_paths"`
	IgnorePatterns []string       `yaml:"ignore_patterns"`
	DisabledScopes []string       `yaml:"disabled_scopes"`
}

// AcceptedItem represents an accepted finding
type AcceptedItem struct {
	Title        string `yaml:"title"`
	Reason       string `yaml:"reason,omitempty"`
	AcceptedBy   string `yaml:"accepted_by,omitempty"`
	AcceptedDate string `yaml:"accepted_date,omitempty"`
}

// LoadIgnoreConfig loads the ignore configuration from .github/autoengineer-ignore.yaml
// Returns an empty config if the file doesn't exist
func LoadIgnoreConfig() (*IgnoreConfig, error) {
	// Check for both .yaml and .yml extensions
	paths := []string{
		".github/autoengineer-ignore.yaml",
		".github/autoengineer-ignore.yml",
	}

	var configPath string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	// Return empty config if no file found
	if configPath == "" {
		return &IgnoreConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config IgnoreConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetAcceptedTitles returns a map of accepted finding titles for quick lookup
func (c *IgnoreConfig) GetAcceptedTitles() map[string]bool {
	titles := make(map[string]bool)
	for _, item := range c.Accepted {
		titles[item.Title] = true
	}
	return titles
}

// IsScopeDisabled checks if a scope is disabled
func (c *IgnoreConfig) IsScopeDisabled(scope string) bool {
	for _, disabled := range c.DisabledScopes {
		if disabled == scope {
			return true
		}
	}
	return false
}

// MatchesPath checks if a file path matches any ignore patterns
func (c *IgnoreConfig) MatchesPath(path string) bool {
	for _, pattern := range c.IgnorePaths {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check if pattern matches any parent directory
		if matchesGlobPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchesGlobPattern handles ** patterns for directory matching
func matchesGlobPattern(path, pattern string) bool {
	// Simple implementation for common patterns like "examples/*" or "**/testdata/**"
	// This is a simplified version - could be enhanced with a proper glob library
	if len(pattern) > 2 && pattern[0:2] == "**" {
		// Pattern like **/testdata/**
		matched, err := filepath.Match(pattern[2:], path)
		return err == nil && matched
	}
	return false
}
