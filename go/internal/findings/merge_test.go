package findings

import (
	"testing"
)

func TestMerge(t *testing.T) {
	findings1 := []Finding{
		{
			Title:       "Security issue in production environment configuration is insecure",
			Description: "Production configuration lacks proper security settings",
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
		},
		{
			Title:    "Security issue B",
			Severity: SeverityLow,
			Category: CategorySecurity,
		},
	}

	findings2 := []Finding{
		{
			Title:    "Pipeline issue",
			Severity: SeverityMedium,
			Category: CategoryPipeline,
		},
		{
			Title:       "Security issue in production environment configuration needs review",
			Description: "Production configuration requires security review and hardening",
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
		},
	}

	merged := Merge(findings1, findings2)

	// First and fourth findings should merge (similar titles and descriptions, same category)
	// Result: 1 merged security finding + 1 other security finding + 1 pipeline = 3
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
		{Title: "Security issue in production environment configuration is insecure", Category: CategorySecurity},
		{Title: "Security issue in production environment configuration needs fix", Category: CategorySecurity}, // Similar title
		{Title: "Pipeline optimization needed", Category: CategoryPipeline},
	}

	result := deduplicate(findings)

	// First two have similar titles and same category, should be merged
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
		{Title: "Low issue", Severity: SeverityLow},
		{Title: "High issue", Severity: SeverityHigh},
		{Title: "Medium issue", Severity: SeverityMedium},
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

// Test grouping findings across multiple files with similar issues
func TestSemanticGroupingAcrossFiles(t *testing.T) {
	findings := []Finding{
		{
			Title:       "S3 bucket lacks encryption",
			Description: "The S3 bucket does not have encryption enabled",
			Files:       []string{"infra/storage.tf"},
			Category:    CategorySecurity,
			Severity:    SeverityHigh,
		},
		{
			Title:       "S3 bucket missing encryption",
			Description: "Encryption is not configured for the S3 bucket",
			Files:       []string{"infra/backup.tf"},
			Category:    CategorySecurity,
			Severity:    SeverityHigh,
		},
	}

	result := deduplicate(findings)

	// Should merge into one finding with both files
	if len(result) != 1 {
		t.Errorf("expected 1 finding after grouping, got %d", len(result))
	}

	if len(result[0].Files) != 2 {
		t.Errorf("expected merged finding to have 2 files, got %d", len(result[0].Files))
	}
}

// Test that findings with different categories are NOT merged unless very similar
func TestNoMergeDifferentCategories(t *testing.T) {
	findings := []Finding{
		{
			Title:       "IAM role has wildcard permissions",
			Description: "IAM role grants overly broad access",
			Category:    CategorySecurity,
			Severity:    SeverityHigh,
		},
		{
			Title:       "Missing CI/CD configuration",
			Description: "Pipeline needs optimization",
			Category:    CategoryPipeline,
			Severity:    SeverityMedium,
		},
	}

	result := deduplicate(findings)

	// Should NOT merge - different categories and dissimilar content
	if len(result) != 2 {
		t.Errorf("expected 2 findings (different categories, dissimilar), got %d", len(result))
	}
}

// Test that findings with different categories CAN merge if very similar
func TestMergeDifferentCategoriesIfVerySimilar(t *testing.T) {
	findings := []Finding{
		{
			Title:       "MSK cluster uses public subnets instead of private subnets",
			Description: "The MSK cluster is configured with public subnets which poses a security risk",
			Category:    CategorySecurity,
			Severity:    SeverityHigh,
			Files:       []string{"kafka.tf"},
		},
		{
			Title:       "MSK uses public subnets instead of private subnets",
			Description: "MSK cluster should use private subnets for better isolation",
			Category:    CategoryInfra,
			Severity:    SeverityMedium,
			Files:       []string{"kafka.tf"},
		},
	}

	result := deduplicate(findings)

	// Should merge - different categories but very similar content
	if len(result) != 1 {
		t.Errorf("expected 1 finding after merging very similar cross-category findings, got %d", len(result))
	}

	// Should preserve the higher severity
	if len(result) > 0 && result[0].Severity != SeverityHigh {
		t.Errorf("merged finding should have high severity, got %s", result[0].Severity)
	}
}

// Test merging findings with overlapping files
func TestMergeWithOverlappingFiles(t *testing.T) {
	findings := []Finding{
		{
			Title:       "Resources missing required tags",
			Description: "Resources should have environment tags",
			Files:       []string{"infra/main.tf", "infra/network.tf"},
			Category:    CategoryInfra,
			Severity:    SeverityLow,
		},
		{
			Title:       "Resources missing required tags",
			Description: "Resources need proper tagging",
			Files:       []string{"infra/main.tf", "infra/compute.tf"},
			Category:    CategoryInfra,
			Severity:    SeverityLow,
		},
	}

	result := deduplicate(findings)

	// Should merge due to identical titles and overlapping files
	if len(result) != 1 {
		t.Errorf("expected 1 finding after merging, got %d", len(result))
	}

	// Should have all unique files
	if len(result) > 0 && len(result[0].Files) != 3 {
		t.Errorf("expected 3 files in merged finding, got %d", len(result[0].Files))
	}
}

// Test that dissimilar findings in same category are NOT merged
func TestNoMergeDissimilarFindings(t *testing.T) {
	findings := []Finding{
		{
			Title:       "S3 bucket lacks encryption",
			Description: "Encryption not enabled",
			Category:    CategorySecurity,
			Severity:    SeverityHigh,
		},
		{
			Title:       "IAM role has wildcard permissions",
			Description: "IAM role grants overly broad access",
			Category:    CategorySecurity,
			Severity:    SeverityMedium,
		},
	}

	result := deduplicate(findings)

	// Should NOT merge - completely different issues
	if len(result) != 2 {
		t.Errorf("expected 2 findings (dissimilar issues), got %d", len(result))
	}
}

// Test similarity calculation
func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		finding1 Finding
		finding2 Finding
		minScore float64 // Minimum expected similarity
		maxScore float64 // Maximum expected similarity
	}{
		{
			name: "identical findings",
			finding1: Finding{
				Title:          "Security issue",
				Description:    "This is a security problem",
				Recommendation: "Fix it",
				Files:          []string{"file1.tf"},
			},
			finding2: Finding{
				Title:          "Security issue",
				Description:    "This is a security problem",
				Recommendation: "Fix it",
				Files:          []string{"file1.tf"},
			},
			minScore: 0.95,
			maxScore: 1.0,
		},
		{
			name: "similar findings",
			finding1: Finding{
				Title:          "S3 bucket lacks encryption",
				Description:    "The S3 bucket does not have encryption enabled",
				Recommendation: "Enable encryption on the S3 bucket",
				Files:          []string{"storage.tf"},
			},
			finding2: Finding{
				Title:          "S3 bucket missing encryption",
				Description:    "Encryption is not configured for S3 bucket",
				Recommendation: "Configure encryption for the bucket",
				Files:          []string{"backup.tf"},
			},
			minScore: 0.3,
			maxScore: 0.7,
		},
		{
			name: "dissimilar findings",
			finding1: Finding{
				Title:          "S3 bucket lacks encryption",
				Description:    "Encryption not enabled",
				Recommendation: "Enable encryption",
				Files:          []string{"storage.tf"},
			},
			finding2: Finding{
				Title:          "IAM role has wildcard permissions",
				Description:    "IAM role grants broad access",
				Recommendation: "Restrict IAM permissions",
				Files:          []string{"iam.tf"},
			},
			minScore: 0.0,
			maxScore: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateSimilarity(tt.finding1, tt.finding2)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("similarity score %.2f not in expected range [%.2f, %.2f]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// Test token similarity
func TestTokenSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64
	}{
		{
			name:     "identical strings",
			s1:       "The quick brown fox",
			s2:       "The quick brown fox",
			minScore: 0.95,
		},
		{
			name:     "similar strings",
			s1:       "S3 bucket lacks encryption",
			s2:       "S3 bucket missing encryption",
			minScore: 0.6,
		},
		{
			name:     "different strings",
			s1:       "S3 bucket encryption",
			s2:       "IAM role permissions",
			minScore: 0.0,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			minScore: 0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tokenSimilarity(tt.s1, tt.s2)
			if score < tt.minScore {
				t.Errorf("token similarity %.2f below minimum %.2f", score, tt.minScore)
			}
		})
	}
}

// Test file overlap calculation
func TestCalculateFileOverlap(t *testing.T) {
	tests := []struct {
		name     string
		files1   []string
		files2   []string
		expected float64
	}{
		{
			name:     "identical files",
			files1:   []string{"file1.tf", "file2.tf"},
			files2:   []string{"file1.tf", "file2.tf"},
			expected: 1.0,
		},
		{
			name:     "partial overlap",
			files1:   []string{"file1.tf", "file2.tf"},
			files2:   []string{"file2.tf", "file3.tf"},
			expected: 0.5,
		},
		{
			name:     "no overlap",
			files1:   []string{"file1.tf"},
			files2:   []string{"file2.tf"},
			expected: 0.0,
		},
		{
			name:     "empty lists",
			files1:   []string{},
			files2:   []string{},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlap := calculateFileOverlap(tt.files1, tt.files2)
			if overlap != tt.expected {
				t.Errorf("file overlap = %.2f, expected %.2f", overlap, tt.expected)
			}
		})
	}
}

// Test merging preserves higher severity
func TestMergePreservesHighSeverity(t *testing.T) {
	high := Finding{
		Title:       "Critical issue",
		Severity:    SeverityHigh,
		Category:    CategorySecurity,
		Description: "High severity description",
		Files:       []string{"file1.tf"},
	}

	low := Finding{
		Title:       "Critical issue",
		Severity:    SeverityLow,
		Category:    CategorySecurity,
		Description: "Low severity description",
		Files:       []string{"file2.tf"},
	}

	findings := []Finding{low, high}
	result := deduplicate(findings)

	if len(result) != 1 {
		t.Errorf("expected 1 merged finding, got %d", len(result))
	}

	if result[0].Severity != SeverityHigh {
		t.Errorf("merged finding should have high severity, got %s", result[0].Severity)
	}
}

func TestMergeWithCodeSnippets(t *testing.T) {
	finding1 := Finding{
		Title:       "Security issue in production environment",
		Description: "Production configuration lacks proper security settings",
		Severity:    SeverityHigh,
		Category:    CategorySecurity,
		Files:       []string{"main.tf"},
		CodeSnippets: []CodeSnippet{
			{
				File:      "main.tf",
				StartLine: 10,
				EndLine:   15,
				Code:      "resource \"aws_s3_bucket\" \"example\" {}",
			},
		},
	}

	finding2 := Finding{
		Title:       "Security issue in production environment configuration",
		Description: "Production configuration requires security review",
		Severity:    SeverityHigh,
		Category:    CategorySecurity,
		Files:       []string{"main.tf"},
		CodeSnippets: []CodeSnippet{
			{
				File:      "main.tf",
				StartLine: 20,
				EndLine:   25,
				Code:      "resource \"aws_instance\" \"example\" {}",
			},
		},
	}

	findings := []Finding{finding1, finding2}
	result := deduplicate(findings)

	// Should merge into one finding
	if len(result) != 1 {
		t.Errorf("expected 1 merged finding, got %d", len(result))
	}

	// Should have both code snippets
	if len(result[0].CodeSnippets) != 2 {
		t.Errorf("expected 2 code snippets, got %d", len(result[0].CodeSnippets))
	}
}

func TestMergeDeduplicatesCodeSnippets(t *testing.T) {
	finding1 := Finding{
		Title:       "Security issue",
		Description: "Issue description",
		Severity:    SeverityHigh,
		Category:    CategorySecurity,
		Files:       []string{"main.tf"},
		CodeSnippets: []CodeSnippet{
			{
				File:      "main.tf",
				StartLine: 10,
				EndLine:   15,
				Code:      "resource \"aws_s3_bucket\" \"example\" {}",
			},
		},
	}

	finding2 := Finding{
		Title:       "Security issue",
		Description: "Issue description",
		Severity:    SeverityHigh,
		Category:    CategorySecurity,
		Files:       []string{"main.tf"},
		CodeSnippets: []CodeSnippet{
			{
				File:      "main.tf",
				StartLine: 10,
				EndLine:   15,
				Code:      "resource \"aws_s3_bucket\" \"example\" {}",
			},
		},
	}

	findings := []Finding{finding1, finding2}
	result := deduplicate(findings)

	// Should merge into one finding
	if len(result) != 1 {
		t.Errorf("expected 1 merged finding, got %d", len(result))
	}

	// Should deduplicate identical code snippets
	if len(result[0].CodeSnippets) != 1 {
		t.Errorf("expected 1 code snippet (deduplicated), got %d", len(result[0].CodeSnippets))
	}
}

func TestSnippetKey(t *testing.T) {
	snippet := CodeSnippet{
		File:      "main.tf",
		StartLine: 10,
		EndLine:   15,
		Code:      "example code",
	}

	key := snippetKey(snippet)
	expected := "main.tf:10-15"

	if key != expected {
		t.Errorf("snippetKey() = %q, want %q", key, expected)
	}
}
