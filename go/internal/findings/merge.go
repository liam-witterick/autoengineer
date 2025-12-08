package findings

import (
	"sort"
	"strings"
)

// Merge combines multiple finding arrays and deduplicates them
func Merge(findingArrays ...[]Finding) []Finding {
	// Combine all findings
	var all []Finding
	for _, findings := range findingArrays {
		all = append(all, findings...)
	}

	// Deduplicate by title similarity
	deduplicated := deduplicate(all)

	// Sort by severity
	sort.Sort(BySeverity(deduplicated))

	return deduplicated
}

// deduplicate removes findings with similar titles and consolidates related findings
func deduplicate(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	// Track which findings have been merged
	merged := make(map[int]bool)
	result := make([]Finding, 0, len(findings))

	for i := 0; i < len(findings); i++ {
		if merged[i] {
			continue
		}

		// Start with the current finding
		base := findings[i]
		
		// Find all similar findings to merge
		for j := i + 1; j < len(findings); j++ {
			if merged[j] {
				continue
			}

			// Check if findings should be merged
			if shouldMerge(base, findings[j]) {
				// Merge findings[j] into base
				base = mergeFindings(base, findings[j])
				merged[j] = true
			}
		}

		result = append(result, base)
	}

	return result
}

// shouldMerge determines if two findings should be consolidated
func shouldMerge(a, b Finding) bool {
	// Must be same category
	if a.Category != b.Category {
		return false
	}

	// Calculate similarity score
	score := calculateSimilarity(a, b)
	
	// Merge if similarity is above threshold (0.38 = 38%)
	// Lower threshold allows for more grouping of related findings across different files
	// Balanced to merge related issues while avoiding false positives
	return score >= 0.38
}

// calculateSimilarity computes a similarity score between two findings (0.0 to 1.0)
func calculateSimilarity(a, b Finding) float64 {
	var score float64
	var weights float64

	// 1. Title similarity (weight: 0.35)
	titleSim := tokenSimilarity(a.Title, b.Title)
	score += titleSim * 0.35
	weights += 0.35

	// 2. Description similarity (weight: 0.25) - only if both have descriptions
	if a.Description != "" && b.Description != "" {
		descSim := tokenSimilarity(a.Description, b.Description)
		score += descSim * 0.25
		weights += 0.25
	}

	// 3. Recommendation similarity (weight: 0.15) - only if both have recommendations
	if a.Recommendation != "" && b.Recommendation != "" {
		recSim := tokenSimilarity(a.Recommendation, b.Recommendation)
		score += recSim * 0.15
		weights += 0.15
	}

	// 4. File overlap (weight: 0.25) - only if both have files
	if len(a.Files) > 0 && len(b.Files) > 0 {
		fileOverlap := calculateFileOverlap(a.Files, b.Files)
		score += fileOverlap * 0.25
		weights += 0.25
	}

	if weights == 0 {
		return 0.0
	}

	return score / weights
}

// tokenSimilarity calculates Jaccard similarity between two strings
func tokenSimilarity(s1, s2 string) float64 {
	if s1 == "" && s2 == "" {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Normalize and tokenize
	tokens1 := tokenize(strings.ToLower(s1))
	tokens2 := tokenize(strings.ToLower(s2))

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Normalize common variations (simple stemming)
	tokens1 = normalizeTokens(tokens1)
	tokens2 = normalizeTokens(tokens2)

	// Calculate Jaccard similarity
	intersection := 0
	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}

	for _, token := range tokens2 {
		if set1[token] {
			intersection++
		}
	}

	union := len(tokens1) + len(tokens2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// normalizeTokens applies simple normalization to tokens (basic stemming)
func normalizeTokens(tokens []string) []string {
	normalized := make([]string, len(tokens))
	for i, token := range tokens {
		normalized[i] = normalizeToken(token)
	}
	return normalized
}

// normalizeToken applies basic normalization rules
func normalizeToken(token string) string {
	// Common suffixes to remove (simple stemming)
	suffixes := []string{"ing", "ed", "s", "es"}
	
	for _, suffix := range suffixes {
		if len(token) > len(suffix)+2 && strings.HasSuffix(token, suffix) {
			return token[:len(token)-len(suffix)]
		}
	}
	
	return token
}

// tokenize splits a string into meaningful tokens (words)
func tokenize(s string) []string {
	// Split on whitespace and common punctuation
	replacer := strings.NewReplacer(
		",", " ",
		".", " ",
		":", " ",
		";", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"/", " ",
		"-", " ",
	)
	s = replacer.Replace(s)

	// Split and filter empty tokens
	parts := strings.Fields(s)
	tokens := make([]string, 0, len(parts))
	for _, token := range parts {
		if len(token) > 1 { // Skip single-char tokens
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// calculateFileOverlap calculates the overlap ratio between two file lists
func calculateFileOverlap(files1, files2 []string) float64 {
	if len(files1) == 0 && len(files2) == 0 {
		return 1.0
	}
	if len(files1) == 0 || len(files2) == 0 {
		return 0.0
	}

	// Create set of files
	set1 := make(map[string]bool)
	for _, f := range files1 {
		set1[f] = true
	}

	// Count overlapping files
	overlap := 0
	for _, f := range files2 {
		if set1[f] {
			overlap++
		}
	}

	// Use the maximum of the two lengths for the denominator
	maxLen := len(files1)
	if len(files2) > maxLen {
		maxLen = len(files2)
	}

	return float64(overlap) / float64(maxLen)
}

// mergeFindings consolidates two findings into one, combining their files
func mergeFindings(a, b Finding) Finding {
	// Use the finding with higher severity as the base
	base := a
	other := b
	if severityValue(b.Severity) < severityValue(a.Severity) {
		base = b
		other = a
	}

	// Merge files (deduplicate)
	fileSet := make(map[string]bool)
	for _, f := range base.Files {
		fileSet[f] = true
	}
	for _, f := range other.Files {
		fileSet[f] = true
	}

	mergedFiles := make([]string, 0, len(fileSet))
	for f := range fileSet {
		mergedFiles = append(mergedFiles, f)
	}

	// Sort files for consistency
	sort.Strings(mergedFiles)

	// Merge descriptions if different and both non-empty
	mergedDesc := base.Description
	if other.Description != "" && other.Description != base.Description {
		// Append additional context if descriptions differ
		if !strings.Contains(base.Description, other.Description) {
			mergedDesc = base.Description + " " + other.Description
		}
	}

	// Merge recommendations similarly
	mergedRec := base.Recommendation
	if other.Recommendation != "" && other.Recommendation != base.Recommendation {
		if !strings.Contains(base.Recommendation, other.Recommendation) {
			mergedRec = base.Recommendation + " " + other.Recommendation
		}
	}

	return Finding{
		ID:             base.ID, // Keep the ID of the higher severity finding
		Category:       base.Category,
		Title:          base.Title,
		Severity:       base.Severity,
		Description:    mergedDesc,
		Recommendation: mergedRec,
		Files:          mergedFiles,
	}
}

// severityValue returns a numeric value for severity (lower is more severe)
func severityValue(severity string) int {
	switch severity {
	case SeverityHigh:
		return 0
	case SeverityMedium:
		return 1
	case SeverityLow:
		return 2
	default:
		return 3
	}
}

// CountBySeverity returns counts of findings by severity level
func CountBySeverity(findings []Finding) (high, medium, low int) {
	for _, f := range findings {
		switch f.Severity {
		case SeverityHigh:
			high++
		case SeverityMedium:
			medium++
		case SeverityLow:
			low++
		}
	}
	return
}

// CountByCategory returns counts of findings by category
func CountByCategory(findings []Finding) (security, pipeline, infra int) {
	for _, f := range findings {
		switch f.Category {
		case CategorySecurity:
			security++
		case CategoryPipeline:
			pipeline++
		case CategoryInfra:
			infra++
		}
	}
	return
}
