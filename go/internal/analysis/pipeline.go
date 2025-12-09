package analysis

import (
	"context"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// PipelineAnalyzer performs CI/CD pipeline-focused analysis
type PipelineAnalyzer struct {
	BaseAnalyzer
}

// NewPipelineAnalyzer creates a new pipeline analyzer
func NewPipelineAnalyzer(base BaseAnalyzer) *PipelineAnalyzer {
	return &PipelineAnalyzer{BaseAnalyzer: base}
}

// Scope returns the scope name
func (a *PipelineAnalyzer) Scope() string {
	return "pipeline"
}

// Run executes the pipeline analysis
func (a *PipelineAnalyzer) Run(ctx context.Context) ([]findings.Finding, error) {
	prompt := `Review the infrastructure code in this repo with a CI/CD PIPELINE focus. Output ONLY a JSON array.

PIPELINE FOCUS AREAS:
- GitHub Actions: Deprecated actions, missing version pins, inefficient workflows
- Caching: Missing or misconfigured cache strategies
- Build optimization: Unnecessary steps, missing parallelization, slow builds
- Workflow triggers: Overly broad triggers, missing path filters
- Secrets handling: Insecure secret injection, missing environment protection
- Reusability: Duplicated workflow logic that could be consolidated
- Artifact management: Missing retention policies, oversized artifacts

Format:
[{"category": "pipeline", "title": "string", "severity": "high|medium|low", "description": "string", "recommendation": "string", "files": ["path/to/file"], "code_snippets": [{"file": "path/to/file", "start_line": 10, "end_line": 20, "code": "snippet text"}]}]

Rules:
- category: Must be "pipeline"
- severity: high, medium, or low (lowercase)
- title: concise, under 80 chars
- files: relative paths from repo root
- code_snippets: Optional but recommended. Include up to 2 concise snippets per finding that show the problem. Each snippet should include file, start_line, end_line, and the exact code. Keep each snippet under 20 lines and escape backticks if present.
- Focus ONLY on CI/CD pipeline issues
- Skip issues documented as TODOs` + a.ExistingContext + a.ExtraContext

	results, err := a.Client.RunAnalysis(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Ensure correct category
	for i := range results {
		results[i].Category = findings.CategoryPipeline
	}

	return results, nil
}
