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
	
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
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
