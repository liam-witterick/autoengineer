package config

import (
	"testing"
)

func TestScannerConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *ScannerConfig
		scanner  string
		expected bool
	}{
		{
			name:     "explicitly enabled",
			config:   &ScannerConfig{Enabled: []string{"aikido"}},
			scanner:  "aikido",
			expected: true,
		},
		{
			name:     "not in enabled list",
			config:   &ScannerConfig{Enabled: []string{"aikido"}},
			scanner:  "checkov",
			expected: false,
		},
		{
			name:     "empty config",
			config:   &ScannerConfig{},
			scanner:  "checkov",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsEnabled(tt.scanner)
			if result != tt.expected {
				t.Errorf("IsEnabled(%s) = %v, expected %v", tt.scanner, result, tt.expected)
			}
		})
	}
}

func TestScannerConfigIsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *ScannerConfig
		scanner  string
		expected bool
	}{
		{
			name:     "explicitly disabled",
			config:   &ScannerConfig{Disabled: []string{"checkov"}},
			scanner:  "checkov",
			expected: true,
		},
		{
			name:     "not in disabled list",
			config:   &ScannerConfig{Disabled: []string{"checkov"}},
			scanner:  "trivy",
			expected: false,
		},
		{
			name:     "empty config",
			config:   &ScannerConfig{},
			scanner:  "checkov",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsDisabled(tt.scanner)
			if result != tt.expected {
				t.Errorf("IsDisabled(%s) = %v, expected %v", tt.scanner, result, tt.expected)
			}
		})
	}
}

func TestLoadScannerConfigNoFile(t *testing.T) {
	// Change to a temp directory where no config exists
	cfg, err := LoadScannerConfig()
	if err != nil {
		t.Fatalf("Expected no error with missing config file, got %v", err)
	}
	
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	
	// Should return empty config
	if len(cfg.Enabled) != 0 {
		t.Errorf("Expected empty enabled list, got %v", cfg.Enabled)
	}
	if len(cfg.Disabled) != 0 {
		t.Errorf("Expected empty disabled list, got %v", cfg.Disabled)
	}
}
