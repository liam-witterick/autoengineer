package analysis

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/liam-witterick/autoengineer/go/internal/copilot"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
	"github.com/liam-witterick/autoengineer/go/internal/issues"
)

// Analyzer defines the interface for running scoped analyses
type Analyzer interface {
	Run(ctx context.Context) ([]findings.Finding, error)
	Scope() string
}

// BaseAnalyzer provides common functionality for all analyzers
type BaseAnalyzer struct {
	Client          *copilot.Client
	ExtraContext    string
	ExistingContext string
}

// GenerateID generates a unique ID for a finding
// Note: Uses MD5 for ID generation (non-security purpose)
func GenerateID(prefix, title string, files []string) string {
	filesStr := strings.Join(files, ",")
	// MD5 is sufficient for generating short, non-cryptographic IDs
	hash := md5.Sum([]byte(title + filesStr))
	return fmt.Sprintf("%s%x", prefix, hash[:4])
}

// EnsureIDs ensures all findings have valid IDs
func EnsureIDs(findings []findings.Finding, prefix string) []findings.Finding {
	for i := range findings {
		if findings[i].ID == "" {
			findings[i].ID = GenerateID(prefix, findings[i].Title, findings[i].Files)
		}
	}
	return findings
}

// BuildExistingContext creates a context string from existing issues to add to the analysis prompt
func BuildExistingContext(existingIssues []issues.SearchResult) string {
	if len(existingIssues) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nSKIP these already-tracked issues (they already have GitHub issues):\n")
	for _, issue := range existingIssues {
		sb.WriteString(fmt.Sprintf("- \"%s\" (Issue #%d)\n", issue.Title, issue.Number))
	}
	sb.WriteString("Do NOT report findings that match these existing issues.\n")
	return sb.String()
}
