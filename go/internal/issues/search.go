package issues

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SearchResult represents a search match
type SearchResult struct {
	Number int
	Title  string
	Body   string
	Labels []string
}

// labelStruct is used for parsing label JSON responses
type labelStruct struct {
	Name string `json:"name"`
}

// extractLabelNames converts label structs to label name strings
func extractLabelNames(labels []labelStruct) []string {
	labelNames := make([]string, len(labels))
	for i, label := range labels {
		labelNames[i] = label.Name
	}
	return labelNames
}

// FindByTitle searches for an issue by title similarity
func (c *Client) FindByTitle(ctx context.Context, title string) (*SearchResult, error) {
	// Use first 50 chars for fuzzy matching
	searchTitle := title
	if len(searchTitle) > 50 {
		searchTitle = searchTitle[:50]
	}
	
	// Clean title for search (remove special chars)
	searchTitle = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return -1
	}, searchTitle)

	query := fmt.Sprintf("\"%s\" in:title repo:%s/%s state:open", searchTitle, c.owner, c.repo)
	encodedQuery := url.QueryEscape(query)
	
	var result struct {
		Items []struct {
			Number int           `json:"number"`
			Title  string        `json:"title"`
			Body   string        `json:"body"`
			Labels []labelStruct `json:"labels"`
		} `json:"items"`
	}

	err := c.apiClient.Get(fmt.Sprintf("search/issues?q=%s", encodedQuery), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to search for issue: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	return &SearchResult{
		Number: result.Items[0].Number,
		Title:  result.Items[0].Title,
		Body:   result.Items[0].Body,
		Labels: extractLabelNames(result.Items[0].Labels),
	}, nil
}

// IssueExists checks if an issue exists by title
func (c *Client) IssueExists(ctx context.Context, title string) (bool, string, error) {
	// Search by title
	result, err := c.FindByTitle(ctx, title)
	if err != nil {
		return false, "", err
	}
	if result != nil {
		return true, "title", nil
	}

	return false, "", nil
}

// GetIssueLabels returns the labels for a specific issue
func (c *Client) GetIssueLabels(ctx context.Context, issueNumber int) ([]string, error) {
	var labels []labelStruct

	err := c.apiClient.Get(fmt.Sprintf("repos/%s/%s/issues/%d/labels", c.owner, c.repo, issueNumber), &labels)
	if err != nil {
		return nil, fmt.Errorf("failed to get labels for issue #%d: %w", issueNumber, err)
	}

	return extractLabelNames(labels), nil
}

// HasDelegatedLabel checks if an issue has the delegated label
func (c *Client) HasDelegatedLabel(ctx context.Context, issueNumber int) (bool, error) {
	labels, err := c.GetIssueLabels(ctx, issueNumber)
	if err != nil {
		return false, err
	}

	for _, label := range labels {
		if label == DelegatedLabel {
			return true, nil
		}
	}

	return false, nil
}

// ListOpenIssues returns all open issues with the autoengineer label
func (c *Client) ListOpenIssues(ctx context.Context) ([]SearchResult, error) {
	query := fmt.Sprintf("repo:%s/%s state:open label:%s", c.owner, c.repo, c.label)
	encodedQuery := url.QueryEscape(query)
	
	var result struct {
		Items []struct {
			Number int           `json:"number"`
			Title  string        `json:"title"`
			Body   string        `json:"body"`
			Labels []labelStruct `json:"labels"`
		} `json:"items"`
	}

	err := c.apiClient.Get(fmt.Sprintf("search/issues?q=%s", encodedQuery), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to search for issues: %w", err)
	}

	issues := make([]SearchResult, len(result.Items))
	for i, item := range result.Items {
		issues[i] = SearchResult{
			Number: item.Number,
			Title:  item.Title,
			Body:   item.Body,
			Labels: extractLabelNames(item.Labels),
		}
	}

	return issues, nil
}
