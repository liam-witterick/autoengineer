package copilot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

// Client wraps interactions with the GitHub Copilot CLI
type Client struct {
	// Path to copilot binary (defaults to "copilot" in PATH)
	BinaryPath string
}

// Deduplicate implements the findings.Deduplicator interface
func (c *Client) Deduplicate(ctx context.Context, findingsToProcess []findings.Finding) ([]findings.Finding, error) {
	return c.DeduplicateFindings(ctx, findingsToProcess)
}

// NewClient creates a new Copilot client
func NewClient() *Client {
	return &Client{
		BinaryPath: "copilot",
	}
}

// Check verifies that the copilot CLI is available
func (c *Client) Check() error {
	cmd := exec.Command(c.BinaryPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copilot CLI not found: %w", err)
	}
	return nil
}

// RunAnalysis runs a copilot analysis with the given prompt and parses JSON output
func (c *Client) RunAnalysis(ctx context.Context, prompt string) ([]findings.Finding, error) {
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-i", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("copilot command failed: %w (stderr: %s)", err, stderr.String())
	}

	// Extract JSON from markdown code block
	jsonStr := extractJSON(stdout.String())
	if jsonStr == "" {
		return []findings.Finding{}, nil
	}

	var results []findings.Finding
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("failed to parse copilot JSON output: %w", err)
	}

	return results, nil
}

// RunFix runs copilot suggest to provide local fix suggestions interactively
func (c *Client) RunFix(ctx context.Context, prompt string) error {
	// Use "gh copilot suggest" for interactive local fixes
	cmd := exec.CommandContext(ctx, "gh", "copilot", "suggest", prompt)
	
	// Connect to stdin/stdout/stderr so user can interact
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// RunDelegate delegates a fix to the Copilot coding agent (cloud)
func (c *Client) RunDelegate(ctx context.Context, prompt string) error {
	// Prepend /delegate to the prompt
	delegatePrompt := "/delegate " + prompt
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-i", delegatePrompt)
	
	// Run in background - capture output for error reporting
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("delegation failed: %w (stderr: %s)", err, stderr.String())
		}
		return err
	}
	
	return nil
}

// DeduplicateFindings uses Copilot to intelligently deduplicate and merge related findings
func (c *Client) DeduplicateFindings(ctx context.Context, findings []findings.Finding) ([]findings.Finding, error) {
	if len(findings) <= 1 {
		return findings, nil
	}

	// Marshal findings to JSON for the prompt
	findingsJSON, err := json.MarshalIndent(findings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal findings: %w", err)
	}

	prompt := "You are a security and infrastructure analysis expert. Review the following findings and deduplicate them intelligently.\n\n" +
		"RULES FOR DEDUPLICATION:\n" +
		"1. Merge findings that describe the SAME ISSUE across multiple files (e.g., \"container running as root\" in different YAML files)\n" +
		"2. Merge findings with similar titles in the same category (e.g., \"S3 bucket encryption missing\" and \"Missing encryption for S3 bucket\")\n" +
		"3. When merging, combine the Files arrays to show all affected files\n" +
		"4. Keep the first finding's ID, Title, Description, Recommendation, Category, and Severity\n" +
		"5. DO NOT merge findings from different categories (security vs pipeline vs infra)\n" +
		"6. DO NOT merge completely unrelated issues even if in the same category\n" +
		"7. Preserve findings that are distinct issues\n\n" +
		"Return ONLY a JSON array of deduplicated findings with the same structure. Each finding should have:\n" +
		"- id: string\n" +
		"- category: string (security, pipeline, or infra)\n" +
		"- title: string\n" +
		"- severity: string (high, medium, or low)\n" +
		"- description: string\n" +
		"- recommendation: string\n" +
		"- files: array of strings (merged from related findings)\n\n" +
		"Input findings:\n" +
		string(findingsJSON) + "\n\n" +
		"Output the deduplicated findings as a JSON array in a json code block."

	return c.RunAnalysis(ctx, prompt)
}

// extractJSON extracts JSON content from markdown code blocks
func extractJSON(output string) string {
	// Look for ```json ... ``` blocks
	re := regexp.MustCompile("(?s)```json\\s*\n(.*?)\n```")
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: try to find JSON array directly
	scanner := bufio.NewScanner(strings.NewReader(output))
	var jsonLines []string
	inJSON := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "[" {
			inJSON = true
			jsonLines = []string{trimmed}
		} else if inJSON {
			jsonLines = append(jsonLines, line)
			if trimmed == "]" {
				break
			}
		}
	}

	if len(jsonLines) > 0 {
		return strings.Join(jsonLines, "\n")
	}

	return ""
}
