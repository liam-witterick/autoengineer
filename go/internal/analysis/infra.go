package analysis

import (
	"context"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// InfraAnalyzer performs infrastructure-focused analysis
type InfraAnalyzer struct {
	BaseAnalyzer
}

// NewInfraAnalyzer creates a new infrastructure analyzer
func NewInfraAnalyzer(base BaseAnalyzer) *InfraAnalyzer {
	return &InfraAnalyzer{BaseAnalyzer: base}
}

// Scope returns the scope name
func (a *InfraAnalyzer) Scope() string {
	return "infra"
}

// Run executes the infrastructure analysis
func (a *InfraAnalyzer) Run(ctx context.Context) ([]findings.Finding, error) {
	prompt := `Review the infrastructure code in this repo with an INFRASTRUCTURE focus. Output ONLY a JSON array.

INFRASTRUCTURE FOCUS AREAS:
- Terraform/OpenTofu: Unpinned module versions, missing state locking, deprecated syntax
- Resource configuration: Missing tags, improper naming conventions, hardcoded values
- Cost optimization: Oversized resources, missing auto-scaling, unused resources
- Kubernetes: Missing resource limits, improper replica counts, missing health checks
- Helm charts: Hardcoded values, missing templating, version inconsistencies
- State management: Backend configuration issues, missing remote state
- Module structure: Poor separation of concerns, missing outputs, undocumented variables

Format:
[{"category": "infra", "title": "string", "severity": "high|medium|low", "description": "string", "recommendation": "string", "files": ["path/to/file"]}]

Rules:
- category: Must be "infra"
- severity: high, medium, or low (lowercase)
- title: concise, under 80 chars
- files: relative paths from repo root
- Focus ONLY on infrastructure issues
- Skip issues documented as TODOs` + a.ExistingContext + a.ExtraContext

	results, err := a.Client.RunAnalysis(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Ensure correct category
	for i := range results {
		results[i].Category = findings.CategoryInfra
	}

	return results, nil
}
