package analysis

import (
	"context"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// SecurityAnalyzer performs security-focused analysis
type SecurityAnalyzer struct {
	BaseAnalyzer
}

// NewSecurityAnalyzer creates a new security analyzer
func NewSecurityAnalyzer(base BaseAnalyzer) *SecurityAnalyzer {
	return &SecurityAnalyzer{BaseAnalyzer: base}
}

// Scope returns the scope name
func (a *SecurityAnalyzer) Scope() string {
	return "security"
}

// Run executes the security analysis
func (a *SecurityAnalyzer) Run(ctx context.Context) ([]findings.Finding, error) {
	prompt := `Review the infrastructure code in this repo with a SECURITY focus. Output ONLY a JSON array.

SECURITY FOCUS AREAS:
- IAM/RBAC policies: Over-permissive roles, missing least-privilege, wildcard permissions
- Network security: Open security groups (0.0.0.0/0), public subnets, exposed ports
- Secrets management: Hardcoded credentials, API keys, tokens in code or configs
- Encryption: Unencrypted storage, missing TLS/SSL, weak cipher suites
- Container security: Running as root, missing security contexts, privileged containers
- Compliance gaps: Missing audit logging, untagged resources

Format:
[{"category": "security", "title": "string", "severity": "high|medium|low", "description": "string", "recommendation": "string", "files": ["path/to/file"], "code_snippets": [{"file": "path/to/file", "start_line": 10, "end_line": 20, "code": "snippet text"}]}]

Rules:
- category: Must be "security"
- severity: high, medium, or low (lowercase)
- title: concise, under 80 chars
- files: relative paths from repo root
- code_snippets: Optional but recommended. Include up to 2 concise snippets per finding that best illustrate the issue. Each snippet should specify file, start_line, end_line, and the exact code. Keep each snippet under 20 lines and escape backticks if present.
- Focus ONLY on security issues
- Skip issues documented as TODOs` + a.ExistingContext + a.ExtraContext

	results, err := a.Client.RunAnalysis(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Ensure correct category
	for i := range results {
		results[i].Category = findings.CategorySecurity
	}

	return results, nil
}
