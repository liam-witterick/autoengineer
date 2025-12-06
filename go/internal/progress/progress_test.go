package progress

import (
	"bytes"
	"testing"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker("Testing", 10)
	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}
	if tracker.total != 10 {
		t.Errorf("Expected total to be 10, got %d", tracker.total)
	}
	if tracker.current != 0 {
		t.Errorf("Expected current to be 0, got %d", tracker.current)
	}
	if !tracker.enabled {
		t.Error("Expected tracker to be enabled")
	}
}

func TestNewTrackerZeroTotal(t *testing.T) {
	tracker := NewTracker("Testing", 0)
	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}
	if tracker.total != 1 {
		t.Errorf("Expected total to be 1 (minimum), got %d", tracker.total)
	}
}

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner("Loading...")
	if spinner == nil {
		t.Fatal("Expected spinner to be created")
	}
	if !spinner.enabled {
		t.Error("Expected spinner to be enabled")
	}
}

func TestTrackerIncrement(t *testing.T) {
	tracker := NewTracker("Testing", 5)
	
	tracker.Increment()
	if tracker.current != 1 {
		t.Errorf("Expected current to be 1, got %d", tracker.current)
	}
	
	tracker.Increment()
	if tracker.current != 2 {
		t.Errorf("Expected current to be 2, got %d", tracker.current)
	}
}

func TestTrackerDisable(t *testing.T) {
	tracker := NewTracker("Testing", 5)
	tracker.Disable()
	
	if tracker.enabled {
		t.Error("Expected tracker to be disabled")
	}
	
	// Should not panic when disabled
	tracker.Increment()
	tracker.SetDescription("New description")
	tracker.Finish()
}

func TestNewScopeTracker(t *testing.T) {
	scopes := []string{"security", "pipeline", "infra"}
	tracker := NewScopeTracker(scopes)
	
	if tracker == nil {
		t.Fatal("Expected scope tracker to be created")
	}
	if tracker.total != 3 {
		t.Errorf("Expected total to be 3, got %d", tracker.total)
	}
	if tracker.done != 0 {
		t.Errorf("Expected done to be 0, got %d", tracker.done)
	}
	if !tracker.enabled {
		t.Error("Expected tracker to be enabled")
	}
}

func TestScopeTrackerStartComplete(t *testing.T) {
	scopes := []string{"security"}
	tracker := NewScopeTracker(scopes)
	
	// Redirect output to buffer to avoid stderr output during tests
	var buf bytes.Buffer
	tracker.output = &buf
	
	tracker.StartScope("security")
	tracker.CompleteScope("security", 5)
	
	if tracker.done != 1 {
		t.Errorf("Expected done to be 1, got %d", tracker.done)
	}
	
	output := buf.String()
	if output == "" {
		t.Error("Expected output to be written")
	}
}

func TestScopeTrackerFail(t *testing.T) {
	scopes := []string{"pipeline"}
	tracker := NewScopeTracker(scopes)
	
	// Redirect output to buffer
	var buf bytes.Buffer
	tracker.output = &buf
	
	tracker.StartScope("pipeline")
	tracker.FailScope("pipeline", nil)
	
	if tracker.done != 1 {
		t.Errorf("Expected done to be 1, got %d", tracker.done)
	}
}

func TestGetScopeEmoji(t *testing.T) {
	tests := []struct {
		scope    string
		expected string
	}{
		{"security", "🔒"},
		{"pipeline", "⚙️"},
		{"infra", "🏗️"},
		{"infrastructure", "🏗️"},
		{"unknown", "📋"},
	}
	
	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			emoji := getScopeEmoji(tt.scope)
			if emoji != tt.expected {
				t.Errorf("Expected emoji %s for scope %s, got %s", tt.expected, tt.scope, emoji)
			}
		})
	}
}
