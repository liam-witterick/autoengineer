package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
)

const (
	// DelegatedLabel is the label name used to mark issues that have been delegated to Copilot coding agent
	DelegatedLabel = "delegated"
)

// Client handles GitHub issue operations
type Client struct {
	apiClient     *api.RESTClient
	graphqlClient *api.GraphQLClient
	owner         string
	repo          string
	label         string
}

// NewClient creates a new GitHub issues client
func NewClient(owner, repo, label string) (*Client, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub API client: %w", err)
	}

	// Create GraphQL client for operations that require it (like assigning Copilot)
	gqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub GraphQL client: %w", err)
	}

	return &Client{
		apiClient:     client,
		graphqlClient: gqlClient,
		owner:         owner,
		repo:          repo,
		label:         label,
	}, nil
}

// ensureLabelExists is a helper function that creates a label if it doesn't exist
func (c *Client) ensureLabelExists(ctx context.Context, name, description, color string) error {
	// Check if label exists
	var label struct {
		Name string `json:"name"`
	}

	err := c.apiClient.Get(fmt.Sprintf("repos/%s/%s/labels/%s", c.owner, c.repo, name), &label)
	if err == nil {
		// Label exists
		return nil
	}

	// Create label
	labelData := map[string]string{
		"name":        name,
		"description": description,
		"color":       color,
	}

	body, err := json.Marshal(labelData)
	if err != nil {
		return err
	}

	err = c.apiClient.Post(fmt.Sprintf("repos/%s/%s/labels", c.owner, c.repo), bytes.NewReader(body), nil)
	if err != nil {
		// Silently ignore if label already exists (422 status)
		// Other errors are also ignored since label creation is not critical
		return nil
	}

	return nil
}

// EnsureLabel creates the autoengineer label if it doesn't exist
func (c *Client) EnsureLabel(ctx context.Context) error {
	return c.ensureLabelExists(ctx, c.label, "AutoEngineer - Autonomous DevOps maintenance issues", "d4c5f9")
}

// EnsureDelegatedLabel creates the delegated label if it doesn't exist
func (c *Client) EnsureDelegatedLabel(ctx context.Context) error {
	return c.ensureLabelExists(ctx, DelegatedLabel, "Issue has been delegated to Copilot coding agent", "0366d6")
}

// AddDelegatedLabel adds the delegated label to an issue
func (c *Client) AddDelegatedLabel(ctx context.Context, issueNumber int) error {
	// Ensure delegated label exists
	if err := c.EnsureDelegatedLabel(ctx); err != nil {
		return err
	}

	// Add label to issue
	labelData := map[string]interface{}{
		"labels": []string{DelegatedLabel},
	}

	body, err := json.Marshal(labelData)
	if err != nil {
		return err
	}

	err = c.apiClient.Post(
		fmt.Sprintf("repos/%s/%s/issues/%d/labels", c.owner, c.repo, issueNumber),
		bytes.NewReader(body),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to add delegated label to issue #%d: %w", issueNumber, err)
	}

	return nil
}

// AssignCopilot assigns the Copilot coding agent to an issue.
// Assigning 'copilot-swe-agent' as an assignee triggers GitHub Copilot's
// coding agent to work on the issue and create a PR with fixes.
// This uses the GraphQL API because the REST API does not support assigning Copilot.
func (c *Client) AssignCopilot(ctx context.Context, issueNumber int) error {
	// Step 1: Query to get the issue node ID and find copilot-swe-agent actor
	query := `
		query($owner: String!, $name: String!, $issueNumber: Int!) {
			repository(owner: $owner, name: $name) {
				issue(number: $issueNumber) {
					id
				}
				suggestedActors(capabilities: [CAN_BE_ASSIGNED], first: 100) {
					nodes {
						login
						... on Bot { id }
						... on User { id }
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":       c.owner,
		"name":        c.repo,
		"issueNumber": issueNumber,
	}

	var queryResponse struct {
		Repository struct {
			Issue struct {
				ID string `json:"id"`
			} `json:"issue"`
			SuggestedActors struct {
				Nodes []struct {
					Login string `json:"login"`
					ID    string `json:"id"`
				} `json:"nodes"`
			} `json:"suggestedActors"`
		} `json:"repository"`
	}

	err := c.graphqlClient.DoWithContext(ctx, query, variables, &queryResponse)
	if err != nil {
		return fmt.Errorf("failed to query issue and suggested actors: %w", err)
	}

	// Step 2: Find copilot-swe-agent in the suggested actors
	var copilotActorID string
	for _, actor := range queryResponse.Repository.SuggestedActors.Nodes {
		if actor.Login == "copilot-swe-agent" {
			copilotActorID = actor.ID
			break
		}
	}

	if copilotActorID == "" {
		return fmt.Errorf("copilot-swe-agent is not available for this repository. Ensure GitHub Copilot coding agent is enabled for your organization/repository")
	}

	// Step 3: Use GraphQL mutation to assign Copilot to the issue
	mutation := `
		mutation($issueId: ID!, $assigneeIds: [ID!]!) {
			addAssigneesToAssignable(input: {assignableId: $issueId, assigneeIds: $assigneeIds}) {
				assignable {
					... on Issue {
						number
						assignees(first: 10) {
							nodes {
								login
							}
						}
					}
				}
			}
		}
	`

	mutationVariables := map[string]interface{}{
		"issueId":     queryResponse.Repository.Issue.ID,
		"assigneeIds": []string{copilotActorID},
	}

	var mutationResponse struct {
		AddAssigneesToAssignable struct {
			Assignable struct {
				Number    int `json:"number"`
				Assignees struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"assignees"`
			} `json:"assignable"`
		} `json:"addAssigneesToAssignable"`
	}

	err = c.graphqlClient.DoWithContext(ctx, mutation, mutationVariables, &mutationResponse)
	if err != nil {
		return fmt.Errorf("failed to assign copilot-swe-agent to issue #%d: %w", issueNumber, err)
	}

	return nil
}

// CheckCopilotAvailability checks if the Copilot coding agent is available for this repository
func (c *Client) CheckCopilotAvailability(ctx context.Context) (bool, error) {
	// Query to check if copilot-swe-agent is in suggested actors
	query := `
		query($owner: String!, $name: String!) {
			repository(owner: $owner, name: $name) {
				suggestedActors(capabilities: [CAN_BE_ASSIGNED], first: 100) {
					nodes {
						login
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner": c.owner,
		"name":  c.repo,
	}

	var response struct {
		Repository struct {
			SuggestedActors struct {
				Nodes []struct {
					Login string `json:"login"`
				} `json:"nodes"`
			} `json:"suggestedActors"`
		} `json:"repository"`
	}

	err := c.graphqlClient.DoWithContext(ctx, query, variables, &response)
	if err != nil {
		return false, fmt.Errorf("failed to query suggested actors: %w", err)
	}

	// Check if copilot-swe-agent is available
	for _, actor := range response.Repository.SuggestedActors.Nodes {
		if actor.Login == "copilot-swe-agent" {
			return true, nil
		}
	}

	return false, nil
}

// CreateIssue creates a GitHub issue from a finding
func (c *Client) CreateIssue(ctx context.Context, finding findings.Finding) (int, error) {
	emoji := severityEmoji(finding.Severity)
	title := fmt.Sprintf("%s %s", emoji, finding.Title)

	body := formatIssueBody(finding)

	issueData := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": []string{c.label},
	}

	bodyBytes, err := json.Marshal(issueData)
	if err != nil {
		return 0, err
	}

	var result struct {
		Number int `json:"number"`
	}

	err = c.apiClient.Post(fmt.Sprintf("repos/%s/%s/issues", c.owner, c.repo), bytes.NewReader(bodyBytes), &result)
	if err != nil {
		return 0, fmt.Errorf("failed to create issue: %w", err)
	}

	return result.Number, nil
}

// formatIssueBody formats the issue body from a finding
func formatIssueBody(finding findings.Finding) string {
	priority := "Unknown"
	switch finding.Severity {
	case findings.SeverityHigh:
		priority = "High"
	case findings.SeverityMedium:
		priority = "Medium"
	case findings.SeverityLow:
		priority = "Low"
	}

	filesStr := ""
	for _, file := range finding.Files {
		filesStr += "- `" + file + "`\n"
	}

	body := fmt.Sprintf(`## Summary
%s

## Recommendation
%s

## Priority
%s

## Files
%s`,
		finding.Description,
		finding.Recommendation,
		priority,
		filesStr,
	)

	// Add code snippets if available
	if len(finding.CodeSnippets) > 0 {
		body += "\n\n## Code References\n"
		for _, snippet := range finding.CodeSnippets {
			// Format header with file and line numbers
			header := fmt.Sprintf("### `%s`", snippet.File)
			if snippet.StartLine > 0 {
				if snippet.EndLine > 0 && snippet.EndLine != snippet.StartLine {
					header += fmt.Sprintf(" (lines %d-%d)", snippet.StartLine, snippet.EndLine)
				} else {
					header += fmt.Sprintf(" (line %d)", snippet.StartLine)
				}
			}
			body += header + "\n"
			
			// Detect language from file extension for syntax highlighting
			lang := detectLanguage(snippet.File)
			body += fmt.Sprintf("```%s\n%s\n```\n\n", lang, snippet.Code)
		}
	}

	body += "\n---\n*Generated by [AutoEngineer](https://github.com/liam-witterick/autoengineer)*"
	
	return body
}

// detectLanguage detects the programming language from a file extension
func detectLanguage(filename string) string {
	// Extract file extension
	lastDot := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			lastDot = i
			break
		}
	}
	
	if lastDot == -1 {
		return ""
	}
	
	ext := filename[lastDot+1:]
	
	// Map common extensions to language identifiers
	languageMap := map[string]string{
		"tf":         "hcl",
		"hcl":        "hcl",
		"go":         "go",
		"py":         "python",
		"js":         "javascript",
		"ts":         "typescript",
		"yaml":       "yaml",
		"yml":        "yaml",
		"json":       "json",
		"sh":         "bash",
		"bash":       "bash",
		"Dockerfile": "dockerfile",
		"java":       "java",
		"c":          "c",
		"cpp":        "cpp",
		"cs":         "csharp",
		"rb":         "ruby",
		"php":        "php",
		"rs":         "rust",
		"kt":         "kotlin",
		"swift":      "swift",
	}
	
	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	
	return ""
}

// severityEmoji returns the emoji for a severity level
func severityEmoji(severity string) string {
	switch severity {
	case findings.SeverityHigh:
		return "🔴"
	case findings.SeverityMedium:
		return "🟡"
	case findings.SeverityLow:
		return "🟢"
	default:
		return "⚪"
	}
}
