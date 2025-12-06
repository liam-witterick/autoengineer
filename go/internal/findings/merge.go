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

	// Group and merge related findings
	merged := groupAndMerge(all)

	// Sort by severity
	sort.Sort(BySeverity(merged))

	return merged
}

// groupAndMerge consolidates closely related findings based on semantic similarity,
// category, affected files, and recommendations
func groupAndMerge(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	var result []Finding
	used := make(map[int]bool)

	for i, finding := range findings {
		if used[i] {
			continue
		}

		// Start a new group with this finding
		group := finding
		used[i] = true

		// Find all related findings
		for j := i + 1; j < len(findings); j++ {
			if used[j] {
				continue
			}

			other := findings[j]
			if shouldMerge(group, other) {
				// Merge files from related findings
				group.Files = mergeFiles(group.Files, other.Files)
				used[j] = true
			}
		}

		result = append(result, group)
	}

	return result
}

// shouldMerge determines if two findings should be merged based on multiple criteria
func shouldMerge(a, b Finding) bool {
	// Must be in the same category
	if a.Category != b.Category {
		return false
	}

	// Check title similarity using multiple approaches
	titleSimilar := areTitlesSimilar(a.Title, b.Title)
	
	// Check if they share common files
	hasCommonFiles := haveCommonFiles(a.Files, b.Files)
	
	// Check description/recommendation similarity
	descSimilar := areDescriptionsSimilar(a.Description, b.Description)
	recSimilar := areRecommendationsSimilar(a.Recommendation, b.Recommendation)

	// Merge if:
	// 1. Titles are very similar (regardless of files)
	// 2. Same category + similar descriptions + share files
	// 3. Same category + similar recommendations + share files
	if titleSimilar {
		return true
	}

	if hasCommonFiles && (descSimilar || recSimilar) {
		return true
	}

	return false
}

// areTitlesSimilar checks if two titles are semantically similar
func areTitlesSimilar(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	// Exact match
	if a == b {
		return true
	}

	// First 50 chars match (legacy behavior)
	prefixA, prefixB := a, b
	if len(prefixA) > 50 {
		prefixA = prefixA[:50]
	}
	if len(prefixB) > 50 {
		prefixB = prefixB[:50]
	}
	if prefixA == prefixB {
		return true
	}

	// Extract significant words
	wordsA := extractWords(a)
	wordsB := extractWords(b)
	
	// Don't merge if either title has very few significant words (too generic)
	// This prevents overly broad matches like "Security issue" matching everything
	if len(wordsA) < 3 || len(wordsB) < 3 {
		return false
	}

	// Check for substantial overlap (75% or more of shorter title)
	return calculateOverlapScore(a, b) >= 0.75
}

// areDescriptionsSimilar checks if descriptions indicate the same issue
func areDescriptionsSimilar(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == "" || b == "" {
		return false
	}

	// Extract key terms and check for overlap
	return calculateOverlapScore(a, b) >= 0.6
}

// areRecommendationsSimilar checks if recommendations are similar
func areRecommendationsSimilar(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if a == "" || b == "" {
		return false
	}

	// Check for substantial overlap in recommendations
	return calculateOverlapScore(a, b) >= 0.6
}

// calculateOverlapScore computes a similarity score based on word overlap
func calculateOverlapScore(a, b string) float64 {
	if a == b {
		return 1.0
	}

	wordsA := extractWords(a)
	wordsB := extractWords(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	// Count common words
	common := 0
	for word := range wordsA {
		if wordsB[word] {
			common++
		}
	}

	// Score based on shorter set (to handle one being a subset of the other)
	minLen := len(wordsA)
	if len(wordsB) < minLen {
		minLen = len(wordsB)
	}

	return float64(common) / float64(minLen)
}

// extractWords extracts significant words from text (excluding common words)
func extractWords(text string) map[string]bool {
	// Common words to ignore
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	}

	words := make(map[string]bool)
	for _, word := range strings.Fields(text) {
		// Clean word
		word = strings.ToLower(strings.Trim(word, ".,;:!?()[]{}\"'"))
		if len(word) > 2 && !stopWords[word] {
			words[word] = true
		}
	}

	return words
}

// haveCommonFiles checks if two findings share any files
func haveCommonFiles(filesA, filesB []string) bool {
	if len(filesA) == 0 || len(filesB) == 0 {
		return false
	}

	fileSet := make(map[string]bool)
	for _, file := range filesA {
		fileSet[file] = true
	}

	for _, file := range filesB {
		if fileSet[file] {
			return true
		}
	}

	return false
}

// mergeFiles combines two file lists, removing duplicates
func mergeFiles(filesA, filesB []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, file := range filesA {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}

	for _, file := range filesB {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}

	return result
}

// deduplicate removes findings with similar titles (legacy function, replaced by groupAndMerge)
func deduplicate(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	seen := make(map[string]bool)
	result := make([]Finding, 0, len(findings))

	for _, finding := range findings {
		// Use first 50 chars of lowercase title as key for fuzzy matching
		title := strings.ToLower(finding.Title)
		if len(title) > 50 {
			title = title[:50]
		}

		if !seen[title] {
			seen[title] = true
			result = append(result, finding)
		}
	}

	return result
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
