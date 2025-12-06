package scanner

import (
	"context"
	"testing"

	"github.com/liam-witterick/infra-review/go/internal/config"
)

func TestCheckovScanner(t *testing.T) {
	scanner := NewCheckovScanner()
	
	if scanner.Name() != "checkov" {
		t.Errorf("Expected name 'checkov', got '%s'", scanner.Name())
	}
	
	if scanner.Type() != TypeLocal {
		t.Errorf("Expected type 'local', got '%s'", scanner.Type())
	}
	
	// Note: We can't test IsInstalled() or Run() reliably in CI
	// as Checkov may not be installed
}

func TestTrivyScanner(t *testing.T) {
	scanner := NewTrivyScanner()
	
	if scanner.Name() != "trivy" {
		t.Errorf("Expected name 'trivy', got '%s'", scanner.Name())
	}
	
	if scanner.Type() != TypeLocal {
		t.Errorf("Expected type 'local', got '%s'", scanner.Type())
	}
	
	// Note: We can't test IsInstalled() or Run() reliably in CI
	// as Trivy may not be installed
}

func TestManager(t *testing.T) {
	cfg := &config.ScannerConfig{}
	mgr := NewManager(cfg)
	
	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}
	
	// Test detection
	statuses := mgr.DetectScanners()
	
	if len(statuses) < 2 {
		t.Errorf("Expected at least 2 scanners, got %d", len(statuses))
	}
	
	// Verify scanner names
	foundCheckov := false
	foundTrivy := false
	
	for _, status := range statuses {
		if status.Name == "checkov" {
			foundCheckov = true
		}
		if status.Name == "trivy" {
			foundTrivy = true
		}
	}
	
	if !foundCheckov {
		t.Error("Expected to find checkov scanner")
	}
	if !foundTrivy {
		t.Error("Expected to find trivy scanner")
	}
}

func TestManagerWithDisabledScanners(t *testing.T) {
	cfg := &config.ScannerConfig{
		Disabled: []string{"checkov"},
	}
	mgr := NewManager(cfg)
	
	statuses := mgr.DetectScanners()
	
	for _, status := range statuses {
		if status.Name == "checkov" {
			if status.Enabled {
				t.Error("Expected checkov to be disabled")
			}
			// Reason will be "not installed" if not installed, or "disabled in config" if installed but disabled
			if status.Installed && status.Reason != "disabled in config" {
				t.Errorf("Expected reason 'disabled in config' for installed scanner, got '%s'", status.Reason)
			} else if !status.Installed && status.Reason != "not installed" {
				t.Errorf("Expected reason 'not installed' for non-installed scanner, got '%s'", status.Reason)
			}
		}
	}
}

func TestManagerRunAllWithNoScanners(t *testing.T) {
	// Disable all scanners
	cfg := &config.ScannerConfig{
		Disabled: []string{"checkov", "trivy"},
	}
	mgr := NewManager(cfg)
	
	ctx := context.Background()
	findings, statuses := mgr.RunAll(ctx, "all")
	
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings with all scanners disabled, got %d", len(findings))
	}
	
	for _, status := range statuses {
		if status.Enabled {
			t.Errorf("Expected all scanners to be disabled, but %s is enabled", status.Name)
		}
	}
}
