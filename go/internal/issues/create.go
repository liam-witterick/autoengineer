package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	graphqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub GraphQL client: %w", err)
	}

	return &Client{
		apiClient:     client,
		graphqlClient: graphqlClient,
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
// Assigning 'copilot' as an assignee is the mechanism that triggers GitHub Copilot's
// coding agent to work on the issue and create a PR with fixes.
func (c *Client) AssignCopilot(ctx context.Context, issueNumber int) error {
	// Step 1: Get the issue node ID using GraphQL
	var issueQuery struct {
		Repository struct {
			Issue struct {
				Id string
			} `graphql:"issue(number: $issueNumber)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	issueVariables := map[string]interface{}{
		"owner":       c.owner,
		"repo":        c.repo,
		"issueNumber": issueNumber,
	}

	err := c.graphqlClient.Query("IssueNodeId", &issueQuery, issueVariables)
	if err != nil {
		return fmt.Errorf("failed to get issue node ID for issue #%d: %w", issueNumber, err)
	}

	issueNodeId := issueQuery.Repository.Issue.Id
	if issueNodeId == "" {
		return fmt.Errorf("issue #%d not found", issueNumber)
	}

	// Step 2: Find Copilot's actor ID from assignable users
	var assignableQuery struct {
		Repository struct {
			AssignableUsers struct {
				Nodes []struct {
					Id    string
					Login string
				}
			} `graphql:"assignableUsers(query: $query, first: $first)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	assignableVariables := map[string]interface{}{
		"owner": c.owner,
		"repo":  c.repo,
		"query": "copilot",
		"first": 10,
	}

	err = c.graphqlClient.Query("AssignableUsers", &assignableQuery, assignableVariables)
	if err != nil {
		return fmt.Errorf("failed to query assignable users: %w", err)
	}

	// Look for copilot user (handle variations like copilot, copilot-swe-agent, github-copilot[bot])
	var copilotActorId string
	for _, user := range assignableQuery.Repository.AssignableUsers.Nodes {
		if strings.Contains(strings.ToLower(user.Login), "copilot") {
			copilotActorId = user.Id
			break
		}
	}

	if copilotActorId == "" {
		return fmt.Errorf("copilot user not found in assignable users for repository %s/%s - ensure Copilot coding agent is enabled for this repository", c.owner, c.repo)
	}

	// Step 3: Assign Copilot using the addAssigneesToAssignable mutation
	var mutation struct {
		AddAssigneesToAssignable struct {
			ClientMutationId string
		} `graphql:"addAssigneesToAssignable(input: $input)"`
	}

	mutationVariables := map[string]interface{}{
		"input": map[string]interface{}{
			"assignableId": issueNodeId,
			"assigneeIds":  []string{copilotActorId},
		},
	}

	err = c.graphqlClient.Mutate("AssignCopilot", &mutation, mutationVariables)
	if err != nil {
		return fmt.Errorf("failed to assign copilot to issue #%d: %w", issueNumber, err)
	}

	return nil
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

	return fmt.Sprintf(`## Summary
%s

## Recommendation
%s

## Priority
%s

## Files
%s

---
*Generated by [AutoEngineer](https://github.com/liam-witterick/autoengineer) · %s*`,
		finding.Description,
		finding.Recommendation,
		priority,
		filesStr,
		finding.ID,
	)
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
