package findings

import (
	"testing"
)

func TestMerge(t *testing.T) {
	findings1 := []Finding{
		{ID: "SEC-001", Title: "Security issue in production environment configuration is insecure", Severity: SeverityHigh},
		{ID: "SEC-002", Title: "Security issue B", Severity: SeverityLow},
	}

	findings2 := []Finding{
		{ID: "PIPE-001", Title: "Pipeline issue", Severity: SeverityMedium},
		{ID: "SEC-003", Title: "Security issue in production environment configuration needs review", Severity: SeverityHigh}, // Similar to SEC-001 (first 50 chars match)
	}

	merged := Merge(findings1, findings2)

	// Should deduplicate similar titles (first 50 chars)
	// Both start with "security issue in production environment configur"
	if len(merged) != 3 {
		t.Errorf("expected 3 findings after merge, got %d", len(merged))
	}

	// Should be sorted by severity (high, medium, low)
	if merged[0].Severity != SeverityHigh {
		t.Errorf("first finding should be high severity, got %s", merged[0].Severity)
	}
	if merged[len(merged)-1].Severity != SeverityLow {
		t.Errorf("last finding should be low severity, got %s", merged[len(merged)-1].Severity)
	}
}

func TestDeduplicate(t *testing.T) {
	findings := []Finding{
		{ID: "SEC-001", Title: "Security issue in production environment configuration is insecure"},
		{ID: "SEC-002", Title: "Security issue in production environment configuration needs fix"}, // Similar title (first 50 chars same)
		{ID: "PIPE-001", Title: "Pipeline optimization needed"},
	}

	result := deduplicate(findings)

	// First two have similar titles (first 50 chars), should keep only first one
	if len(result) != 2 {
		t.Errorf("expected 2 findings after deduplication, got %d", len(result))
	}
}

func TestCountBySeverity(t *testing.T) {
	findings := []Finding{
		{Severity: SeverityHigh},
		{Severity: SeverityHigh},
		{Severity: SeverityMedium},
		{Severity: SeverityLow},
	}

	high, medium, low := CountBySeverity(findings)

	if high != 2 {
		t.Errorf("expected 2 high severity, got %d", high)
	}
	if medium != 1 {
		t.Errorf("expected 1 medium severity, got %d", medium)
	}
	if low != 1 {
		t.Errorf("expected 1 low severity, got %d", low)
	}
}

func TestCountByCategory(t *testing.T) {
	findings := []Finding{
		{Category: CategorySecurity},
		{Category: CategorySecurity},
		{Category: CategoryPipeline},
		{Category: CategoryInfra},
	}

	security, pipeline, infra := CountByCategory(findings)

	if security != 2 {
		t.Errorf("expected 2 security findings, got %d", security)
	}
	if pipeline != 1 {
		t.Errorf("expected 1 pipeline finding, got %d", pipeline)
	}
	if infra != 1 {
		t.Errorf("expected 1 infra finding, got %d", infra)
	}
}

func TestBySeverity(t *testing.T) {
	findings := []Finding{
		{ID: "1", Severity: SeverityLow},
		{ID: "2", Severity: SeverityHigh},
		{ID: "3", Severity: SeverityMedium},
	}

	// Test the sort interface
	bs := BySeverity(findings)
	if bs.Len() != 3 {
		t.Errorf("expected length 3, got %d", bs.Len())
	}

	// Test Less - high should be less than medium
	if !bs.Less(1, 2) { // High (idx 1) should be less than Medium (idx 2)
		t.Error("high severity should sort before medium")
	}
}
