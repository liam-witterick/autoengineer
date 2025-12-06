package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ScannerConfig represents the scanner configuration
type ScannerConfig struct {
	Enabled  []string      `yaml:"enabled"`
	Disabled []string      `yaml:"disabled"`
	Aikido   *AikidoConfig `yaml:"aikido,omitempty"`
}

// AikidoConfig represents Aikido-specific configuration
type AikidoConfig struct {
	APIKeyEnv string `yaml:"api_key_env"`
}

// FullConfig represents the complete autoengineer.yaml structure
type FullConfig struct {
	Scanners *ScannerConfig `yaml:"scanners"`
}

// LoadScannerConfig loads the scanner configuration from .github/autoengineer.yaml
// Returns a default config if the file doesn't exist
func LoadScannerConfig() (*ScannerConfig, error) {
	// Check for both .yaml and .yml extensions
	paths := []string{
		".github/autoengineer.yaml",
		".github/autoengineer.yml",
	}

	var configPath string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	// Return default config if no file found
	if configPath == "" {
		return &ScannerConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse the full config structure
	var fullConfig FullConfig

	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return nil, err
	}

	// Return scanner config or empty if not present
	if fullConfig.Scanners != nil {
		return fullConfig.Scanners, nil
	}

	return &ScannerConfig{}, nil
}

// IsEnabled checks if a scanner is explicitly enabled
func (c *ScannerConfig) IsEnabled(name string) bool {
	for _, enabled := range c.Enabled {
		if enabled == name {
			return true
		}
	}
	return false
}

// IsDisabled checks if a scanner is explicitly disabled
func (c *ScannerConfig) IsDisabled(name string) bool {
	for _, disabled := range c.Disabled {
		if disabled == name {
			return true
		}
	}
	return false
}
