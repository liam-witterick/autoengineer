package findings

import (
	"testing"
)

func TestMerge(t *testing.T) {
	findings1 := []Finding{
		{ID: "SEC-001", Title: "Security issue in production environment configuration is insecure", Severity: SeverityHigh, Category: CategorySecurity},
		{ID: "SEC-002", Title: "Security issue B", Severity: SeverityLow, Category: CategorySecurity},
	}

	findings2 := []Finding{
		{ID: "PIPE-001", Title: "Pipeline issue", Severity: SeverityMedium, Category: CategoryPipeline},
		{ID: "SEC-003", Title: "Security issue in production environment configuration needs review", Severity: SeverityHigh, Category: CategorySecurity}, // Similar to SEC-001 (first 50 chars match)
	}

	merged := Merge(findings1, findings2)

	// Should merge SEC-001 and SEC-003 (similar titles, same category)
	// SEC-002 should remain separate (too short/generic to merge based on title alone)
	// PIPE-001 should remain separate (different category)
	// Expected: 3 findings total
	if len(merged) != 3 {
		t.Errorf("expected 3 findings after merge, got %d", len(merged))
		for i, f := range merged {
			t.Logf("Finding %d: %s (Category: %s, Severity: %s)", i, f.Title, f.Category, f.Severity)
		}
	}

	// Should be sorted by severity (high, medium, low)
	if merged[0].Severity != SeverityHigh {
		t.Errorf("first finding should be high severity, got %s", merged[0].Severity)
	}
	if merged[len(merged)-1].Severity != SeverityLow {
		t.Errorf("last finding should be low severity, got %s", merged[len(merged)-1].Severity)
	}
}

func TestGroupAndMergeSameCategorySimilarTitles(t *testing.T) {
	findings := []Finding{
		{
			ID:       "SEC-001",
			Title:    "Missing encryption for S3 bucket data",
			Category: CategorySecurity,
			Files:    []string{"infra/s3.tf"},
		},
		{
			ID:       "SEC-002",
			Title:    "S3 bucket data missing encryption",
			Category: CategorySecurity,
			Files:    []string{"infra/s3-backup.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should merge due to similar titles in same category
	if len(result) != 1 {
		t.Errorf("expected 1 finding after grouping, got %d", len(result))
	}

	// Should combine files from both findings
	if len(result[0].Files) != 2 {
		t.Errorf("expected 2 files in merged finding, got %d", len(result[0].Files))
	}
}

func TestGroupAndMergeSameIssueMultipleFiles(t *testing.T) {
	findings := []Finding{
		{
			ID:          "SEC-001",
			Title:       "Container running as root user",
			Category:    CategorySecurity,
			Description: "Container is running with root privileges which is a security risk",
			Files:       []string{"k8s/deployment-api.yaml"},
		},
		{
			ID:          "SEC-002",
			Title:       "Container running as root",
			Category:    CategorySecurity,
			Description: "Container runs with root privileges posing security risk",
			Files:       []string{"k8s/deployment-web.yaml"},
		},
		{
			ID:          "SEC-003",
			Title:       "Root user in container",
			Category:    CategorySecurity,
			Description: "Container is running with root privileges which is a security risk",
			Files:       []string{"k8s/deployment-worker.yaml"},
		},
	}

	result := groupAndMerge(findings)

	// Should merge all three due to similar descriptions and same category
	if len(result) != 1 {
		t.Errorf("expected 1 finding after grouping, got %d", len(result))
	}

	// Should have all three files
	if len(result[0].Files) != 3 {
		t.Errorf("expected 3 files in merged finding, got %d", len(result[0].Files))
	}
}

func TestGroupAndMergeSameFilesSimilarRecommendations(t *testing.T) {
	findings := []Finding{
		{
			ID:             "INFRA-001",
			Title:          "Terraform module version not pinned",
			Category:       CategoryInfra,
			Recommendation: "Pin module version to specific release",
			Files:          []string{"infra/main.tf"},
		},
		{
			ID:             "INFRA-002",
			Title:          "Module version unpinned",
			Category:       CategoryInfra,
			Recommendation: "Pin the module version to a specific release",
			Files:          []string{"infra/main.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should merge due to same file and similar recommendations
	if len(result) != 1 {
		t.Errorf("expected 1 finding after grouping, got %d", len(result))
	}
}

func TestGroupAndMergeDifferentCategories(t *testing.T) {
	findings := []Finding{
		{
			ID:       "SEC-001",
			Title:    "Missing encryption",
			Category: CategorySecurity,
			Files:    []string{"file.tf"},
		},
		{
			ID:       "INFRA-001",
			Title:    "Missing encryption",
			Category: CategoryInfra,
			Files:    []string{"file.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should NOT merge - different categories
	if len(result) != 2 {
		t.Errorf("expected 2 findings (different categories), got %d", len(result))
	}
}

func TestGroupAndMergeDifferentIssues(t *testing.T) {
	findings := []Finding{
		{
			ID:          "SEC-001",
			Title:       "Missing encryption for data at rest",
			Category:    CategorySecurity,
			Description: "S3 bucket does not have encryption enabled",
			Files:       []string{"s3.tf"},
		},
		{
			ID:          "SEC-002",
			Title:       "Security group allows public access",
			Category:    CategorySecurity,
			Description: "Security group allows ingress from 0.0.0.0/0",
			Files:       []string{"sg.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should NOT merge - completely different issues
	if len(result) != 2 {
		t.Errorf("expected 2 findings (different issues), got %d", len(result))
	}
}

func TestGroupAndMergeOverlappingFilesNoOtherSimilarity(t *testing.T) {
	findings := []Finding{
		{
			ID:          "SEC-001",
			Title:       "Missing encryption",
			Category:    CategorySecurity,
			Description: "Data is not encrypted at rest",
			Files:       []string{"infra/security.tf", "infra/main.tf"},
		},
		{
			ID:          "SEC-002",
			Title:       "Open security group",
			Category:    CategorySecurity,
			Description: "Security group allows all traffic",
			Files:       []string{"infra/main.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should NOT merge - share a file but completely different issues
	if len(result) != 2 {
		t.Errorf("expected 2 findings (different issues despite shared file), got %d", len(result))
	}
}

func TestGroupAndMergeShortGenericTitles(t *testing.T) {
	findings := []Finding{
		{
			ID:       "SEC-001",
			Title:    "Security issue A",
			Category: CategorySecurity,
			Files:    []string{"file1.tf"},
		},
		{
			ID:       "SEC-002",
			Title:    "Security issue B",
			Category: CategorySecurity,
			Files:    []string{"file2.tf"},
		},
	}

	result := groupAndMerge(findings)

	// Should NOT merge - titles are too short/generic (< 3 significant words each)
	if len(result) != 2 {
		t.Errorf("expected 2 findings (short generic titles), got %d", len(result))
	}
}

func TestAreTitlesSimilar(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "exact match",
			a:        "Missing encryption",
			b:        "Missing encryption",
			expected: true,
		},
		{
			name:     "case insensitive match",
			a:        "Missing Encryption",
			b:        "missing encryption",
			expected: true,
		},
		{
			name:     "first 50 chars match",
			a:        "Security issue in production environment configuration is insecure",
			b:        "Security issue in production environment configuration needs review",
			expected: true,
		},
		{
			name:     "high word overlap",
			a:        "Container running as root user",
			b:        "Container running as root",
			expected: true,
		},
		{
			name:     "different order but similar",
			a:        "S3 bucket missing encryption",
			b:        "Missing encryption for S3 bucket",
			expected: true,
		},
		{
			name:     "completely different",
			a:        "Missing encryption",
			b:        "Security group open to public",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := areTitlesSimilar(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("areTitlesSimilar(%q, %q) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCalculateOverlapScore(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		minScore float64 // minimum expected score
	}{
		{
			name:     "identical texts",
			a:        "container running as root",
			b:        "container running as root",
			minScore: 1.0,
		},
		{
			name:     "high overlap",
			a:        "container running as root user",
			b:        "container running as root",
			minScore: 0.75,
		},
		{
			name:     "medium overlap",
			a:        "missing encryption for data",
			b:        "data missing proper security",
			minScore: 0.3,
		},
		{
			name:     "no overlap",
			a:        "missing encryption",
			b:        "security group open",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateOverlapScore(tt.a, tt.b)
			if score < tt.minScore {
				t.Errorf("calculateOverlapScore(%q, %q) = %v, expected at least %v", tt.a, tt.b, score, tt.minScore)
			}
		})
	}
}

func TestHaveCommonFiles(t *testing.T) {
	tests := []struct {
		name     string
		filesA   []string
		filesB   []string
		expected bool
	}{
		{
			name:     "exact match",
			filesA:   []string{"main.tf"},
			filesB:   []string{"main.tf"},
			expected: true,
		},
		{
			name:     "partial overlap",
			filesA:   []string{"main.tf", "vars.tf"},
			filesB:   []string{"main.tf", "outputs.tf"},
			expected: true,
		},
		{
			name:     "no overlap",
			filesA:   []string{"main.tf"},
			filesB:   []string{"other.tf"},
			expected: false,
		},
		{
			name:     "empty first list",
			filesA:   []string{},
			filesB:   []string{"main.tf"},
			expected: false,
		},
		{
			name:     "empty second list",
			filesA:   []string{"main.tf"},
			filesB:   []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := haveCommonFiles(tt.filesA, tt.filesB)
			if result != tt.expected {
				t.Errorf("haveCommonFiles(%v, %v) = %v, expected %v", tt.filesA, tt.filesB, result, tt.expected)
			}
		})
	}
}

func TestMergeFiles(t *testing.T) {
	tests := []struct {
		name     string
		filesA   []string
		filesB   []string
		expected int // expected number of unique files
	}{
		{
			name:     "no duplicates",
			filesA:   []string{"a.tf", "b.tf"},
			filesB:   []string{"c.tf", "d.tf"},
			expected: 4,
		},
		{
			name:     "with duplicates",
			filesA:   []string{"a.tf", "b.tf"},
			filesB:   []string{"b.tf", "c.tf"},
			expected: 3,
		},
		{
			name:     "all duplicates",
			filesA:   []string{"a.tf", "b.tf"},
			filesB:   []string{"a.tf", "b.tf"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeFiles(tt.filesA, tt.filesB)
			if len(result) != tt.expected {
				t.Errorf("mergeFiles(%v, %v) returned %d files, expected %d", tt.filesA, tt.filesB, len(result), tt.expected)
			}
		})
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
