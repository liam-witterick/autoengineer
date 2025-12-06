package interactive

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/liam-witterick/autoengineer/go/internal/copilot"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
	"github.com/liam-witterick/autoengineer/go/internal/issues"
)

// ActionableItem represents something that can be acted upon (finding or existing issue)
type ActionableItem struct {
	Finding    *findings.Finding
	IssueNum   *int
	IssueTitle string
	IsExisting bool
}

// InteractiveSession manages the interactive prompt flow
type InteractiveSession struct {
	findings      []findings.Finding
	owner         string
	repo          string
	label         string
	issuesClient  *issues.Client
	copilotClient *copilot.Client
	reader        *bufio.Reader
}

// NewSession creates a new interactive session
func NewSession(findings []findings.Finding, owner, repo, label string) (*InteractiveSession, error) {
	client, err := issues.NewClient(owner, repo, label)
	if err != nil {
		return nil, fmt.Errorf("failed to create issues client: %w", err)
	}

	return &InteractiveSession{
		findings:      findings,
		owner:         owner,
		repo:          repo,
		label:         label,
		issuesClient:  client,
		copilotClient: copilot.NewClient(),
		reader:        bufio.NewReader(os.Stdin),
	}, nil
}

// Run starts the interactive prompt loop
func (s *InteractiveSession) Run(ctx context.Context) error {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for {
		fmt.Println()
		fmt.Print("Action: [f]ix, [l]ater, [p]review, [q]uit: ")

		input, err := s.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		action := strings.TrimSpace(strings.ToLower(input))

		switch action {
		case "f", "fix":
			if err := s.handleFix(ctx); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "l", "later":
			if err := s.handleLater(ctx); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "p", "preview":
			s.handlePreview()
		case "q", "quit":
			fmt.Println("\n👋 Exiting. Findings saved to findings.json")
			return nil
		default:
			fmt.Printf("❌ Invalid option: %s (use f, l, p, or q)\n", action)
		}
	}
}

// handleFix handles the [f]ix option
func (s *InteractiveSession) handleFix(ctx context.Context) error {
	// Get existing tracked issues
	fmt.Println("\n🔍 Fetching tracked issues...")

	existingIssues, err := s.getTrackedIssues(ctx)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to fetch tracked issues: %v\n", err)
		existingIssues = []ActionableItem{}
	}

	// Combine existing issues and new findings
	allItems := append(existingIssues, s.convertFindingsToItems()...)

	if len(allItems) == 0 {
		fmt.Println("✅ No items to fix!")
		return nil
	}

	// Display all items
	fmt.Println()
	fmt.Println("📋 Available items to fix:")
	fmt.Println()

	for i, item := range allItems {
		if item.IsExisting {
			fmt.Printf("%d. [TRACKED] %s (Issue #%d)\n", i+1, item.IssueTitle, *item.IssueNum)
		} else {
			emoji := findings.SeverityEmoji(item.Finding.Severity)
			fmt.Printf("%d. %s [NEW] %s\n", i+1, emoji, item.Finding.Title)
		}
	}

	// Get user selection
	fmt.Println()
	fmt.Print("Select items to fix (e.g., 1,3,5 or 'all' or 'cancel'): ")
	selectionInput, err := s.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	selection := strings.TrimSpace(selectionInput)
	if strings.ToLower(selection) == "cancel" {
		fmt.Println("❌ Cancelled")
		return nil
	}

	var selectedItems []ActionableItem
	if strings.ToLower(selection) == "all" {
		selectedItems = allItems
	} else {
		selectedItems, err = s.parseSelection(selection, allItems)
		if err != nil {
			return err
		}
	}

	if len(selectedItems) == 0 {
		fmt.Println("❌ No items selected")
		return nil
	}

	// Ask for delegation method
	fmt.Println()
	fmt.Println("Choose how to fix:")
	fmt.Println("  [l]ocal  - Fix with Copilot CLI (immediate, local changes)")
	fmt.Println("  [c]loud  - Create issue (if needed) + delegate to Copilot coding agent (automated PR)")
	fmt.Print("Method: ")

	methodInput, err := s.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read method: %w", err)
	}

	method := strings.TrimSpace(strings.ToLower(methodInput))

	switch method {
	case "l", "local":
		return s.fixLocal(ctx, selectedItems)
	case "c", "cloud":
		return s.fixCloud(ctx, selectedItems)
	default:
		return fmt.Errorf("invalid method: %s (use l or c)", method)
	}
}

// handleLater creates issues for new findings without fixing them
func (s *InteractiveSession) handleLater(ctx context.Context) error {
	if len(s.findings) == 0 {
		fmt.Println("✅ No new findings to track!")
		return nil
	}

	// Convert findings to items for display
	allItems := s.convertFindingsToItems()

	// Display all items
	fmt.Println()
	fmt.Println("📋 Available items to track:")
	fmt.Println()

	for i, item := range allItems {
		emoji := findings.SeverityEmoji(item.Finding.Severity)
		fmt.Printf("%d. %s [NEW] %s\n", i+1, emoji, item.Finding.Title)
	}

	// Get user selection
	fmt.Println()
	fmt.Print("Select items to track (e.g., 1,3,5 or 'all' or 'cancel'): ")
	selectionInput, err := s.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read selection: %w", err)
	}

	selection := strings.TrimSpace(selectionInput)
	if strings.ToLower(selection) == "cancel" {
		fmt.Println("❌ Cancelled")
		return nil
	}

	var selectedItems []ActionableItem
	if strings.ToLower(selection) == "all" {
		selectedItems = allItems
	} else {
		selectedItems, err = s.parseSelection(selection, allItems)
		if err != nil {
			return err
		}
	}

	if len(selectedItems) == 0 {
		fmt.Println("❌ No items selected")
		return nil
	}

	fmt.Println()
	fmt.Println("📝 Creating GitHub issues for selected findings...")

	// Ensure label exists
	if err := s.issuesClient.EnsureLabel(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to ensure label exists: %v\n", err)
	}

	created := 0
	skipped := 0
	failed := 0

	for _, item := range selectedItems {
		finding := item.Finding
		if finding == nil {
			fmt.Printf("⚠️  Warning: skipping invalid item\n")
			skipped++
			continue
		}

		// Check if issue exists
		exists, matchType, err := s.issuesClient.IssueExists(ctx, finding.ID, finding.Title)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to check for existing issue: %v\n", err)
		}
		if exists {
			fmt.Printf("⏭️  Skipping (exists via %s): %s\n", matchType, finding.Title)
			skipped++
			continue
		}

		fmt.Printf("📝 Creating: %s\n", finding.Title)
		issueNum, err := s.issuesClient.CreateIssue(ctx, *finding)
		if err != nil {
			fmt.Printf("   ❌ Failed: %v\n", err)
			failed++
			continue
		}

		fmt.Printf("   ✅ Created [%s] #%d\n", finding.ID, issueNum)
		created++
	}

	fmt.Println()
	fmt.Printf("📊 Summary: Created=%d, Skipped=%d, Failed=%d\n", created, skipped, failed)

	return nil
}

// handlePreview redisplays the findings summary
func (s *InteractiveSession) handlePreview() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 NEW FINDINGS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	findings.DisplaySummary(s.findings)
	fmt.Println()
	findings.DisplayFindings(s.findings, findings.DetailedDisplayOptions())
}

// getTrackedIssues fetches existing open issues with the autoengineer label
func (s *InteractiveSession) getTrackedIssues(ctx context.Context) ([]ActionableItem, error) {
	results, err := s.issuesClient.ListOpenIssues(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ActionableItem, len(results))
	for i, result := range results {
		issueNum := result.Number
		items[i] = ActionableItem{
			IssueNum:   &issueNum,
			IssueTitle: result.Title,
			IsExisting: true,
		}
	}

	return items, nil
}

// convertFindingsToItems converts findings to actionable items
func (s *InteractiveSession) convertFindingsToItems() []ActionableItem {
	items := make([]ActionableItem, len(s.findings))
	for i := range s.findings {
		items[i] = ActionableItem{
			Finding:    &s.findings[i],
			IsExisting: false,
		}
	}
	return items
}

// parseSelection parses user selection string (e.g., "1,3,5") into selected items
func (s *InteractiveSession) parseSelection(selection string, items []ActionableItem) ([]ActionableItem, error) {
	var selected []ActionableItem
	parts := strings.Split(selection, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}

		if num < 1 || num > len(items) {
			return nil, fmt.Errorf("selection out of range: %d (valid: 1-%d)", num, len(items))
		}

		selected = append(selected, items[num-1])
	}

	return selected, nil
}

// fixLocal delegates to Copilot CLI for local fixes
func (s *InteractiveSession) fixLocal(ctx context.Context, items []ActionableItem) error {
	fmt.Println()
	fmt.Println("🔧 Fixing locally with Copilot CLI...")
	fmt.Println()

	for _, item := range items {
		var prompt string

		if item.IsExisting {
			fmt.Printf("📌 Issue #%d: %s\n", *item.IssueNum, item.IssueTitle)
			prompt = fmt.Sprintf("Fix the issue: %s (see %s/%s#%d for details)",
				item.IssueTitle, s.owner, s.repo, *item.IssueNum)
		} else {
			fmt.Printf("📌 Finding: %s\n", item.Finding.Title)
			prompt = fmt.Sprintf("Fix: %s. %s\nFiles: %s\nRecommendation: %s",
				item.Finding.Title,
				item.Finding.Description,
				strings.Join(item.Finding.Files, ", "),
				item.Finding.Recommendation)
		}

		fmt.Println()
		fmt.Printf("Running: gh copilot suggest...\n")
		fmt.Println()

		if err := s.copilotClient.RunFix(ctx, prompt); err != nil {
			fmt.Printf("⚠️  Warning: copilot suggest failed: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Println()
		fmt.Println("✅ Copilot suggest completed")
		fmt.Println()
	}

	return nil
}

// fixCloud creates issues (if needed) and delegates to Copilot coding agent
func (s *InteractiveSession) fixCloud(ctx context.Context, items []ActionableItem) error {
	fmt.Println()
	fmt.Println("🤖 Delegating to Copilot coding agent...")

	// Ensure labels exist
	if err := s.issuesClient.EnsureLabel(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to ensure label exists: %v\n", err)
	}
	if err := s.issuesClient.EnsureDelegatedLabel(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to ensure delegated label exists: %v\n", err)
	}

	issueNums := []int{}

	for _, item := range items {
		var issueNum int

		if item.IsExisting {
			// Use existing issue
			issueNum = *item.IssueNum
			fmt.Printf("🔧 Using existing issue #%d\n", issueNum)
		} else {
			// Create new issue for finding
			exists, matchType, err := s.issuesClient.IssueExists(ctx, item.Finding.ID, item.Finding.Title)
			if err != nil {
				fmt.Printf("⚠️  Warning: failed to check for existing issue: %v\n", err)
			}
			if exists {
				fmt.Printf("⏭️  Skipping (exists via %s): %s\n", matchType, item.Finding.Title)
				continue
			}

			fmt.Printf("📝 Creating issue: %s\n", item.Finding.Title)
			num, err := s.issuesClient.CreateIssue(ctx, *item.Finding)
			if err != nil {
				fmt.Printf("   ❌ Failed to create issue: %v\n", err)
				continue
			}
			issueNum = num
			fmt.Printf("   ✅ Created issue #%d\n", issueNum)
		}

		issueNums = append(issueNums, issueNum)
	}

	if len(issueNums) == 0 {
		fmt.Println("❌ No issues to delegate")
		return nil
	}

	// Delegate issues
	return s.delegateIssues(ctx, issueNums)
}

// delegateIssues delegates issues to Copilot coding agent
func (s *InteractiveSession) delegateIssues(ctx context.Context, issueNums []int) error {
	for _, issueNum := range issueNums {
		// Check if already delegated
		hasDelegated, err := s.issuesClient.HasDelegatedLabel(ctx, issueNum)
		if err != nil {
			fmt.Printf("   ⚠️  Warning: failed to check delegation status for issue #%d: %v\n", issueNum, err)
		} else if hasDelegated {
			fmt.Printf("⏭️  Skipping issue #%d (already delegated)\n", issueNum)
			continue
		}

		fmt.Printf("🔧 Delegating issue #%d...\n", issueNum)

		// Add delegated label
		if err := s.issuesClient.AddDelegatedLabel(ctx, issueNum); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to add delegated label to issue #%d: %v\n", issueNum, err)
		}

		// Delegate to copilot coding agent
		prompt := fmt.Sprintf("Fix the issue described in %s/%s#%d", s.owner, s.repo, issueNum)

		if err := s.copilotClient.RunDelegate(ctx, prompt); err != nil {
			fmt.Printf("   ⚠️  Warning: delegation failed for issue #%d: %v\n", issueNum, err)
			continue
		}

		fmt.Printf("   ✅ Delegated issue #%d\n", issueNum)
	}

	fmt.Println()
	fmt.Printf("📊 Delegated %d issue(s) to Copilot coding agent\n", len(issueNums))

	return nil
}
