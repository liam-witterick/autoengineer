package analysis

import (
	"context"
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
