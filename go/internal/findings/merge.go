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

// deduplicate removes findings with similar titles
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
