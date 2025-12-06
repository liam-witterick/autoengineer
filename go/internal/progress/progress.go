package progress

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// Tracker manages progress tracking for analysis operations
type Tracker struct {
	bar     *progressbar.ProgressBar
	total   int
	current int
	mu      sync.Mutex
	enabled bool
}

// NewTracker creates a new progress tracker
func NewTracker(description string, total int) *Tracker {
	if total <= 0 {
		total = 1
	}

	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(true),
	)

	return &Tracker{
		bar:     bar,
		total:   total,
		current: 0,
		enabled: true,
	}
}

// NewSpinner creates a spinner for indeterminate progress
func NewSpinner(description string) *Tracker {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(100),
		progressbar.OptionEnableColorCodes(true),
	)

	return &Tracker{
		bar:     bar,
		enabled: true,
	}
}

// Increment increments the progress by one step
func (t *Tracker) Increment() {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.current++
	t.bar.Add(1)
}

// SetDescription updates the progress description
func (t *Tracker) SetDescription(desc string) {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.bar.Describe(desc)
}

// Finish completes the progress bar
func (t *Tracker) Finish() {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Fill to 100% if not already there
	if t.current < t.total {
		t.bar.Add(t.total - t.current)
	}
	t.bar.Finish()
	fmt.Fprintln(os.Stderr) // Add newline after progress bar
}

// Clear clears the progress bar from the terminal
func (t *Tracker) Clear() {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.bar.Clear()
}

// Disable disables progress tracking (useful for testing or non-TTY)
func (t *Tracker) Disable() {
	t.enabled = false
	if t.bar != nil {
		t.bar.Clear()
	}
}

// ScopeTracker tracks progress for multiple scopes
type ScopeTracker struct {
	scopes  map[string]*Tracker
	total   int
	done    int
	mu      sync.Mutex
	output  io.Writer
	enabled bool
}

// NewScopeTracker creates a tracker for multiple scopes
func NewScopeTracker(scopes []string) *ScopeTracker {
	st := &ScopeTracker{
		scopes:  make(map[string]*Tracker),
		total:   len(scopes),
		output:  os.Stderr,
		enabled: true,
	}

	for _, scope := range scopes {
		st.scopes[scope] = nil
	}

	return st
}

// StartScope starts progress tracking for a specific scope
func (st *ScopeTracker) StartScope(scope string) {
	if !st.enabled {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	desc := fmt.Sprintf("[%d/%d] 🔍 Analyzing %s", st.done+1, st.total, scope)
	st.scopes[scope] = NewSpinner(desc)
}

// CompleteScope marks a scope as complete
func (st *ScopeTracker) CompleteScope(scope string, count int) {
	if !st.enabled {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if tracker, exists := st.scopes[scope]; exists && tracker != nil {
		tracker.Finish()
	}

	st.done++

	// Print result
	emoji := getScopeEmoji(scope)
	fmt.Fprintf(st.output, "   ✅ %s %s: %d finding(s)\n", emoji, scope, count)
}

// FailScope marks a scope as failed
func (st *ScopeTracker) FailScope(scope string, err error) {
	if !st.enabled {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if tracker, exists := st.scopes[scope]; exists && tracker != nil {
		tracker.Clear()
	}

	st.done++

	emoji := getScopeEmoji(scope)
	fmt.Fprintf(st.output, "   ❌ %s %s: failed (%v)\n", emoji, scope, err)
}

// Finish completes all scope tracking
func (st *ScopeTracker) Finish() {
	if !st.enabled {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	for _, tracker := range st.scopes {
		if tracker != nil {
			tracker.Clear()
		}
	}
}

// getScopeEmoji returns the emoji for a scope
func getScopeEmoji(scope string) string {
	switch scope {
	case "security":
		return "🔒"
	case "pipeline":
		return "⚙️"
	case "infra", "infrastructure":
		return "🏗️"
	default:
		return "📋"
	}
}
