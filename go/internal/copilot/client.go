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
	"github.com/liam-witterick/autoengineer/go/internal/issues"
)

// Client wraps interactions with the GitHub Copilot CLI
type Client struct {
	// Path to copilot binary (defaults to "copilot" in PATH)
	BinaryPath string
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
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-p", prompt)

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

// RunFix runs copilot interactively to provide local fix suggestions
func (c *Client) RunFix(ctx context.Context, prompt string) error {
	// Use "copilot -i" for interactive local fixes
	// The -i flag enables interactive mode where user can chat with Copilot
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-i")
	
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
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-p", delegatePrompt)
	
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

// RunDeduplication uses Copilot to intelligently deduplicate findings
// It merges related findings (even across categories) and filters out findings
// that match existing tracked issues
func (c *Client) RunDeduplication(ctx context.Context, newFindings []findings.Finding, existingIssues []issues.SearchResult) ([]findings.Finding, error) {
	// If no findings or nothing to deduplicate against, return as-is
	if len(newFindings) == 0 {
		return newFindings, nil
	}

	// Build the deduplication prompt
	prompt := buildDeduplicationPrompt(newFindings, existingIssues)

	// Run copilot analysis with the deduplication prompt
	cmd := exec.CommandContext(ctx, c.BinaryPath, "-p", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If deduplication fails, return error so caller can use original findings
		return newFindings, fmt.Errorf("copilot deduplication failed: %w (stderr: %q)", err, stderr.String())
	}

	// Extract JSON from output
	jsonStr := extractJSON(stdout.String())
	if jsonStr == "" {
		// If we can't parse output, return error - Copilot may not have understood the prompt
		// Include first 200 chars of output for debugging
		output := stdout.String()
		if len(output) > 200 {
			output = output[:200] + "..."
		}
		return newFindings, fmt.Errorf("no JSON output from copilot deduplication (received: %q)", output)
	}

	var deduplicated []findings.Finding
	if err := json.Unmarshal([]byte(jsonStr), &deduplicated); err != nil {
		// If JSON parsing fails, return error so caller can use original findings
		return newFindings, fmt.Errorf("failed to parse deduplication JSON output: %w", err)
	}

	return deduplicated, nil
}

// buildDeduplicationPrompt creates the prompt for Copilot deduplication
func buildDeduplicationPrompt(newFindings []findings.Finding, existingIssues []issues.SearchResult) string {
	var prompt strings.Builder

	prompt.WriteString("You are a deduplication assistant for infrastructure findings. ")
	prompt.WriteString("Your task is to intelligently merge duplicate/related findings and filter out findings that match existing tracked issues.\n\n")

	// Add existing tracked issues if any
	if len(existingIssues) > 0 {
		prompt.WriteString("EXISTING TRACKED ISSUES (already have GitHub issues - remove related findings):\n")
		for _, issue := range existingIssues {
			prompt.WriteString(fmt.Sprintf("- \"%s\" (Issue #%d)\n", issue.Title, issue.Number))
		}
		prompt.WriteString("\n")
	}

	// Add the findings to deduplicate as JSON
	prompt.WriteString("NEW FINDINGS TO DEDUPLICATE:\n")
	findingsJSON, err := json.MarshalIndent(newFindings, "", "  ")
	if err != nil {
		// Fallback to simple representation if JSON marshaling fails
		// Manually build JSON with proper escaping for each field
		prompt.WriteString("[\n")
		for i, f := range newFindings {
			if i > 0 {
				prompt.WriteString(",\n")
			}
			// Marshal individual fields to ensure proper escaping
			// json.Marshal for strings always succeeds, so we can ignore errors
			idJSON, _ := json.Marshal(f.ID)
			titleJSON, _ := json.Marshal(f.Title)
			categoryJSON, _ := json.Marshal(f.Category)
			severityJSON, _ := json.Marshal(f.Severity)
			prompt.WriteString(fmt.Sprintf("  {\"id\": %s, \"title\": %s, \"category\": %s, \"severity\": %s}",
				idJSON, titleJSON, categoryJSON, severityJSON))
		}
		prompt.WriteString("\n]\n\n")
	} else {
		prompt.WriteString(string(findingsJSON))
		prompt.WriteString("\n\n")
	}

	// Add deduplication instructions
	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Merge findings that describe the same underlying issue, even if they're from different categories (security/pipeline/infra)\n")
	prompt.WriteString("2. When merging, keep the finding with the highest severity and combine the file lists (remove duplicates)\n")
	prompt.WriteString("3. Remove any findings that are duplicates or closely related to the existing tracked issues listed above\n")
	prompt.WriteString("4. Keep the ID, category, and severity from the highest severity finding when merging\n")
	prompt.WriteString("5. Combine descriptions and recommendations when merging, separating with '; '\n")
	prompt.WriteString("6. Return ONLY the deduplicated findings as a JSON array in this exact format:\n")
	prompt.WriteString("[{\"id\": \"string\", \"category\": \"string\", \"title\": \"string\", \"severity\": \"string\", ")
	prompt.WriteString("\"description\": \"string\", \"recommendation\": \"string\", \"files\": [\"string\"]}]\n\n")
	prompt.WriteString("Output ONLY the JSON array with no explanation or markdown code blocks.\n")

	return prompt.String()
}

// extractJSON extracts JSON content from markdown code blocks
func extractJSON(output string) string {
	// Try markdown code block first
	re := regexp.MustCompile("(?s)```json\\s*\n(.*?)\n```")
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		candidate := strings.TrimSpace(matches[1])
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}

	// Try to find JSON array starting with [{ (common for array of objects)
	startIdx := strings.Index(output, "[{")
	if startIdx != -1 {
		// Find matching closing bracket using depth tracking
		depth := 0
		for i := startIdx; i < len(output); i++ {
			switch output[i] {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					candidate := output[startIdx : i+1]
					if json.Valid([]byte(candidate)) {
						return candidate
					}
				}
			}
		}
	}

	// Fallback: try to find JSON array directly (line-by-line)
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
		candidate := strings.Join(jsonLines, "\n")
		// Validate before returning
		if json.Valid([]byte(candidate)) {
			return candidate
		}
	}

	return ""
}
