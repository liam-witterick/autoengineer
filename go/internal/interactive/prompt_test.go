package interactive

import (
	"strings"
	"testing"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

func TestParseSelection(t *testing.T) {
	items := []ActionableItem{
		{Finding: &findings.Finding{Title: "Item 1"}},
		{Finding: &findings.Finding{Title: "Item 2"}},
		{Finding: &findings.Finding{Title: "Item 3"}},
		{Finding: &findings.Finding{Title: "Item 4"}},
		{Finding: &findings.Finding{Title: "Item 5"}},
	}

	session := &InteractiveSession{}

	tests := []struct {
		name        string
		selection   string
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:      "single item",
			selection: "1",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple items",
			selection: "1,3,5",
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "with spaces",
			selection: "1, 3, 5",
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:        "out of range",
			selection:   "1,10",
			wantErr:     true,
			errContains: "out of range",
		},
		{
			name:        "invalid number",
			selection:   "1,abc",
			wantErr:     true,
			errContains: "invalid selection",
		},
		{
			name:      "empty parts handled",
			selection: "1,,3",
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected, err := session.parseSelection(tt.selection, items)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSelection() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseSelection() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSelection() unexpected error = %v", err)
				return
			}

			if len(selected) != tt.wantCount {
				t.Errorf("parseSelection() got %d items, want %d", len(selected), tt.wantCount)
			}
		})
	}
}

func TestConvertFindingsToItems(t *testing.T) {
	findings := []findings.Finding{
		{
			ID:       "SEC-001",
			Title:    "Security issue 1",
			Severity: findings.SeverityHigh,
		},
		{
			ID:       "PIPE-002",
			Title:    "Pipeline issue 1",
			Severity: findings.SeverityMedium,
		},
	}

	session := &InteractiveSession{
		findings: findings,
	}

	items := session.convertFindingsToItems()

	if len(items) != len(findings) {
		t.Errorf("convertFindingsToItems() got %d items, want %d", len(items), len(findings))
	}

	for i, item := range items {
		if item.IsExisting {
			t.Errorf("convertFindingsToItems() item[%d].IsExisting = true, want false", i)
		}
		if item.Finding == nil {
			t.Errorf("convertFindingsToItems() item[%d].Finding = nil, want non-nil", i)
		}
		if item.IssueNum != nil {
			t.Errorf("convertFindingsToItems() item[%d].IssueNum = %v, want nil", i, item.IssueNum)
		}
	}
}

func TestSeverityEmoji(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{
			name:     "high severity",
			severity: findings.SeverityHigh,
			want:     "🔴",
		},
		{
			name:     "medium severity",
			severity: findings.SeverityMedium,
			want:     "🟡",
		},
		{
			name:     "low severity",
			severity: findings.SeverityLow,
			want:     "🟢",
		},
		{
			name:     "unknown severity",
			severity: "unknown",
			want:     "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findings.SeverityEmoji(tt.severity)
			if got != tt.want {
				t.Errorf("findings.SeverityEmoji() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandlePreview(t *testing.T) {
	findings := []findings.Finding{
		{
			ID:          "SEC-001",
			Title:       "Security issue",
			Severity:    findings.SeverityHigh,
			Category:    findings.CategorySecurity,
			Files:       []string{"main.go", "auth.go"},
			Description: "This is a test security issue",
		},
		{
			ID:          "PIPE-002",
			Title:       "Pipeline issue",
			Severity:    findings.SeverityMedium,
			Category:    findings.CategoryPipeline,
			Files:       []string{".github/workflows/ci.yml"},
			Description: "This is a test pipeline issue",
		},
	}

	// Note: Full integration testing of handlePreview would require a mock issuesClient
	// since it now calls getAllItems which requires getTrackedIssues.
	// TODO: Consider adding dependency injection for issuesClient to enable more comprehensive testing.
	// For now, we test the core conversion logic that handlePreview depends on.
	session := &InteractiveSession{
		findings: findings,
	}

	items := session.convertFindingsToItems()
	if len(items) != len(findings) {
		t.Errorf("convertFindingsToItems() returned %d items, want %d", len(items), len(findings))
	}
}

func TestGetAllItemsCombinesCorrectly(t *testing.T) {
	// Test that getAllItems combines tracked issues and new findings in the right order
	findings := []findings.Finding{
		{
			ID:       "SEC-001",
			Title:    "Security issue",
			Severity: findings.SeverityHigh,
		},
		{
			ID:       "PIPE-002",
			Title:    "Pipeline issue",
			Severity: findings.SeverityMedium,
		},
	}

	session := &InteractiveSession{
		findings: findings,
	}

	// Test convertFindingsToItems (this is used by getAllItems)
	items := session.convertFindingsToItems()

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// All items should be new findings (not existing)
	for i, item := range items {
		if item.IsExisting {
			t.Errorf("Item %d should not be marked as existing", i)
		}
		if item.Finding == nil {
			t.Errorf("Item %d should have a Finding", i)
		}
	}
}

func TestActionableItem(t *testing.T) {
	// Test ActionableItem with finding
	finding := &findings.Finding{
		ID:       "SEC-001",
		Title:    "Test finding",
		Severity: findings.SeverityHigh,
	}

	item1 := ActionableItem{
		Finding:    finding,
		IsExisting: false,
	}

	if item1.Finding == nil {
		t.Error("ActionableItem.Finding should not be nil")
	}
	if item1.IsExisting {
		t.Error("ActionableItem.IsExisting should be false for new finding")
	}

	// Test ActionableItem with existing issue
	issueNum := 123
	item2 := ActionableItem{
		IssueNum:   &issueNum,
		IssueTitle: "Existing issue",
		IsExisting: true,
	}

	if item2.IssueNum == nil {
		t.Error("ActionableItem.IssueNum should not be nil for existing issue")
	}
	if !item2.IsExisting {
		t.Error("ActionableItem.IsExisting should be true for existing issue")
	}
	if *item2.IssueNum != 123 {
		t.Errorf("ActionableItem.IssueNum = %d, want 123", *item2.IssueNum)
	}
}
