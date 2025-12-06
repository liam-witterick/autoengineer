package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/liam-witterick/autoengineer/go/internal/analysis"
	"github.com/liam-witterick/autoengineer/go/internal/config"
	"github.com/liam-witterick/autoengineer/go/internal/copilot"
	"github.com/liam-witterick/autoengineer/go/internal/findings"
	"github.com/liam-witterick/autoengineer/go/internal/issues"
	"github.com/liam-witterick/autoengineer/go/internal/scanner"
	"github.com/spf13/cobra"
)

const (
	version = "1.0.0"
)

var (
	flagAuto         bool
	flagCreateIssues bool
	flagDelegate     bool
	flagMinSeverity  string
	flagOutput       string
	flagForce        bool
	flagScope        string
	flagCheck        bool
	flagVersion      bool
	flagNoScanners   bool
	flagFast         bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "autoengineer",
		Short: "AutoEngineer - Orchestrating GitHub Copilot for autonomous DevOps maintenance",
		Long: `AutoEngineer v` + version + ` - Orchestrating GitHub Copilot for autonomous DevOps maintenance

AutoEngineer discovers, tracks, and delegates infrastructure and pipeline improvements
powered entirely by GitHub Copilot CLI and Copilot coding agent.

WORKFLOW:
    1. 🔍 DISCOVER  - Copilot CLI analyzes repo, finds issues
    2. 📋 TRACK     - Creates GitHub Issues from findings
    3. 🔧 DELEGATE  - Sends fixes to Copilot CLI (local) or Copilot Coding Agent (cloud)
    4. 🔗 CLOSE     - PRs link back to issues, closing the loop`,
		RunE: run,
	}

	rootCmd.Flags().BoolVar(&flagAuto, "auto", false, "[DEPRECATED] Use --create-issues instead")
	rootCmd.Flags().BoolVar(&flagCreateIssues, "create-issues", false, "Skip prompts and create GitHub issues automatically")
	rootCmd.Flags().BoolVar(&flagDelegate, "delegate", false, "Skip prompts and delegate fixes to Copilot coding agent (requires --create-issues)")
	rootCmd.Flags().StringVar(&flagMinSeverity, "min-severity", "low", "Only action findings at or above this severity level (low|medium|high)")
	rootCmd.Flags().StringVar(&flagOutput, "output", "./findings.json", "Save findings to specified file")
	rootCmd.Flags().BoolVar(&flagForce, "force", false, "Create issues even if duplicates exist")
	rootCmd.Flags().StringVar(&flagScope, "scope", "all", "Run focused analysis (security|pipeline|infra|all)")
	rootCmd.Flags().BoolVar(&flagCheck, "check", false, "Check dependencies and exit")
	rootCmd.Flags().BoolVar(&flagVersion, "version", false, "Show version")
	rootCmd.Flags().BoolVar(&flagNoScanners, "no-scanners", false, "Skip external scanner integration")
	rootCmd.Flags().BoolVar(&flagFast, "fast", false, "Fast mode - skip external scanners (alias for --no-scanners)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if flagVersion {
		fmt.Printf("AutoEngineer v%s\n", version)
		return nil
	}

	if flagCheck {
		return checkDependencies()
	}

	// Handle --auto deprecation
	if flagAuto {
		fmt.Println("⚠️  --auto is deprecated. Use --create-issues instead.")
		flagCreateIssues = true
	}

	// Validate flag combinations
	if flagDelegate && !flagCreateIssues {
		return fmt.Errorf("--delegate requires --create-issues")
	}

	// Validate min-severity
	if flagMinSeverity != "" && !findings.ValidateSeverity(flagMinSeverity) {
		return fmt.Errorf("invalid --min-severity: %s (must be low, medium, or high)", flagMinSeverity)
	}

	// Ensure we're in a git repo
	if !isGitRepo() {
		return fmt.Errorf("not in a git repository")
	}

	ctx := context.Background()

	// Load ignore configuration
	cfg, err := config.LoadIgnoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load ignore config: %w", err)
	}

	// Load scanner configuration
	scannerCfg, err := config.LoadScannerConfig()
	if err != nil {
		return fmt.Errorf("failed to load scanner config: %w", err)
	}

	// Check if scope is disabled
	if flagScope != "all" && cfg.IsScopeDisabled(flagScope) {
		fmt.Printf("⚠️  Scope '%s' is disabled in ignore config\n", flagScope)
		return nil
	}

	// Run analysis
	fmt.Println("\n🔍 Running analysis...")

	// Determine if scanners should run
	skipScanners := flagNoScanners || flagFast

	allFindings, scannerStatuses, err := runAnalysisWithScanners(ctx, flagScope, cfg, scannerCfg, skipScanners)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Display scanner summary
	if !skipScanners && len(scannerStatuses) > 0 {
		displayScannerSummary(scannerStatuses)
	}

	// Filter findings by ignore config
	filtered, ignoredCount := findings.Filter(allFindings, cfg)

	if ignoredCount > 0 {
		fmt.Printf("   Ignored %d finding(s) based on config\n", ignoredCount)
	}

	// Apply severity filtering
	beforeSeverityFilter := len(filtered)
	if flagMinSeverity != "" && flagMinSeverity != findings.SeverityLow {
		filtered = findings.FilterBySeverity(filtered, flagMinSeverity)
		severityFilteredCount := beforeSeverityFilter - len(filtered)
		if severityFilteredCount > 0 {
			fmt.Printf("   Filtered %d finding(s) below %s severity\n", severityFilteredCount, flagMinSeverity)
		}
	}

	// Save findings to file
	if err := saveFindings(filtered, flagOutput); err != nil {
		return fmt.Errorf("failed to save findings: %w", err)
	}

	// Display preview
	displayPreview(filtered, ignoredCount)

	if len(filtered) == 0 {
		fmt.Println("\n✅ No findings to report!")
		return nil
	}

	// Auto mode: create issues automatically
	if flagCreateIssues {
		issueNums, err := createIssuesAuto(ctx, filtered)
		if err != nil {
			return err
		}

		// Delegate to Copilot coding agent if requested
		if flagDelegate && len(issueNums) > 0 {
			return delegateIssues(ctx, issueNums)
		}

		return nil
	}

	// Interactive mode (simplified for now)
	fmt.Println("\n💡 Findings saved to", flagOutput)
	fmt.Println("   Use --create-issues flag to automatically create GitHub issues")

	return nil
}

func checkDependencies() error {
	fmt.Println("\n🔍 Checking dependencies...")
	fmt.Println()

	allOK := true

	// Check copilot
	if checkCommand("copilot") {
		ver := getVersion("copilot", "--version")
		fmt.Printf("   ✅ copilot (%s)\n", ver)
	} else {
		fmt.Println("   ❌ copilot (missing)")
		allOK = false
	}

	// Check gh
	if checkCommand("gh") {
		ver := getVersion("gh", "--version")
		fmt.Printf("   ✅ gh (%s)\n", ver)
	} else {
		fmt.Println("   ❌ gh (missing)")
		allOK = false
	}

	// Check gh auth
	if checkCommand("gh") {
		cmd := exec.Command("gh", "auth", "status")
		if err := cmd.Run(); err == nil {
			fmt.Println("   ✅ gh authenticated")
		} else {
			fmt.Println("   ❌ gh not authenticated (run: gh auth login)")
			allOK = false
		}
	}

	// Check scanners
	fmt.Println()
	fmt.Println("🔍 External Scanners:")

	// Load scanner config
	scannerCfg, err := config.LoadScannerConfig()
	if err != nil {
		fmt.Printf("   ⚠️  Warning: failed to load scanner config: %v\n", err)
		scannerCfg = &config.ScannerConfig{}
	}

	mgr := scanner.NewManager(scannerCfg)
	statuses := mgr.DetectScanners()

	for _, status := range statuses {
		if status.Installed {
			fmt.Printf("   ✅ %s (%s)\n", status.Name, status.Version)
		} else {
			fmt.Printf("   ⏭️  %s (not installed - will be skipped)\n", status.Name)
		}
	}

	fmt.Println()

	if !allOK {
		return fmt.Errorf("some dependencies are missing")
	}

	fmt.Println("✅ All dependencies are installed")
	return nil
}

func checkCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func getVersion(cmd string, args ...string) string {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "installed"
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return "installed"
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func runAnalysis(ctx context.Context, scope string, cfg *config.IgnoreConfig) ([]findings.Finding, error) {
	client := copilot.NewClient()

	base := analysis.BaseAnalyzer{
		Client: client,
	}

	var allFindings []findings.Finding

	switch scope {
	case "security":
		if cfg.IsScopeDisabled("security") {
			fmt.Println("   ⚠️  Security scope is disabled")
			return []findings.Finding{}, nil
		}
		analyzer := analysis.NewSecurityAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			return nil, err
		}
		allFindings = results

	case "pipeline":
		if cfg.IsScopeDisabled("pipeline") {
			fmt.Println("   ⚠️  Pipeline scope is disabled")
			return []findings.Finding{}, nil
		}
		analyzer := analysis.NewPipelineAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			return nil, err
		}
		allFindings = results

	case "infra":
		if cfg.IsScopeDisabled("infra") {
			fmt.Println("   ⚠️  Infra scope is disabled")
			return []findings.Finding{}, nil
		}
		analyzer := analysis.NewInfraAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			return nil, err
		}
		allFindings = results

	case "all":
		// Run all scopes concurrently
		type result struct {
			findings []findings.Finding
			err      error
		}

		secCh := make(chan result, 1)
		pipeCh := make(chan result, 1)
		infraCh := make(chan result, 1)

		// Security
		go func() {
			if cfg.IsScopeDisabled("security") {
				secCh <- result{findings: []findings.Finding{}}
				return
			}
			analyzer := analysis.NewSecurityAnalyzer(base)
			results, err := analyzer.Run(ctx)
			secCh <- result{findings: results, err: err}
		}()

		// Pipeline
		go func() {
			if cfg.IsScopeDisabled("pipeline") {
				pipeCh <- result{findings: []findings.Finding{}}
				return
			}
			analyzer := analysis.NewPipelineAnalyzer(base)
			results, err := analyzer.Run(ctx)
			pipeCh <- result{findings: results, err: err}
		}()

		// Infrastructure
		go func() {
			if cfg.IsScopeDisabled("infra") {
				infraCh <- result{findings: []findings.Finding{}}
				return
			}
			analyzer := analysis.NewInfraAnalyzer(base)
			results, err := analyzer.Run(ctx)
			infraCh <- result{findings: results, err: err}
		}()

		// Collect results
		secResult := <-secCh
		pipeResult := <-pipeCh
		infraResult := <-infraCh

		if secResult.err != nil {
			return nil, fmt.Errorf("security analysis failed: %w", secResult.err)
		}
		if pipeResult.err != nil {
			return nil, fmt.Errorf("pipeline analysis failed: %w", pipeResult.err)
		}
		if infraResult.err != nil {
			return nil, fmt.Errorf("infrastructure analysis failed: %w", infraResult.err)
		}

		allFindings = findings.Merge(secResult.findings, pipeResult.findings, infraResult.findings)

	default:
		return nil, fmt.Errorf("invalid scope: %s (must be security|pipeline|infra|all)", scope)
	}

	return allFindings, nil
}

// runAnalysisWithScanners runs both Copilot analysis and external scanners in parallel
func runAnalysisWithScanners(ctx context.Context, scope string, cfg *config.IgnoreConfig, scannerCfg *config.ScannerConfig, skipScanners bool) ([]findings.Finding, []scanner.ScannerStatus, error) {
	type result struct {
		findings []findings.Finding
		statuses []scanner.ScannerStatus
		err      error
	}

	// Run Copilot analysis and scanners in parallel
	copilotCh := make(chan result, 1)
	scannerCh := make(chan result, 1)

	// Run Copilot analysis
	go func() {
		copilotFindings, err := runAnalysis(ctx, scope, cfg)
		copilotCh <- result{findings: copilotFindings, err: err}
	}()

	// Run scanners if not skipped
	go func() {
		if skipScanners {
			scannerCh <- result{findings: []findings.Finding{}, statuses: []scanner.ScannerStatus{}}
			return
		}

		mgr := scanner.NewManager(scannerCfg)
		scannerFindings, statuses := mgr.RunAll(ctx, scope)
		scannerCh <- result{findings: scannerFindings, statuses: statuses}
	}()

	// Collect results
	copilotResult := <-copilotCh
	scannerResult := <-scannerCh

	if copilotResult.err != nil {
		return nil, nil, copilotResult.err
	}

	// Merge findings from all sources and deduplicate
	allFindings := findings.Merge(copilotResult.findings, scannerResult.findings)

	return allFindings, scannerResult.statuses, nil
}

// displayScannerSummary shows which scanners ran and their results
func displayScannerSummary(statuses []scanner.ScannerStatus) {
	fmt.Println()
	fmt.Println("📊 Scanner Summary:")

	for _, status := range statuses {
		if status.Ran {
			if status.Found > 0 {
				fmt.Printf("   ✅ %s: %d finding(s)\n", status.Name, status.Found)
			} else {
				fmt.Printf("   ✅ %s: no findings\n", status.Name)
			}
		} else if status.Error != nil {
			fmt.Printf("   ⚠️  %s: failed (%v)\n", status.Name, status.Error)
		} else if status.Skipped {
			fmt.Printf("   ⏭️  %s: skipped (%s)\n", status.Name, status.Reason)
		}
	}
}

func saveFindings(allFindings []findings.Finding, outputPath string) error {
	data, err := json.MarshalIndent(allFindings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}

func displayPreview(allFindings []findings.Finding, ignoredCount int) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 NEW FINDINGS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	high, medium, low := findings.CountBySeverity(allFindings)
	security, pipeline, infra := findings.CountByCategory(allFindings)
	total := len(allFindings)

	fmt.Printf("Summary: 🔴 High: %d  🟡 Medium: %d  🟢 Low: %d  (Total: %d)\n", high, medium, low, total)

	if security+pipeline+infra > 0 {
		fmt.Println()
		if security > 0 {
			fmt.Printf("🔒 Security:       %d finding(s)\n", security)
		}
		if pipeline > 0 {
			fmt.Printf("⚙️  Pipeline:       %d finding(s)\n", pipeline)
		}
		if infra > 0 {
			fmt.Printf("🏗️  Infrastructure: %d finding(s)\n", infra)
		}
	}

	if ignoredCount > 0 {
		fmt.Printf("\n⏭️  Ignored:        %d finding(s) (based on ignore config)\n", ignoredCount)
	}

	fmt.Println()

	// Show first few findings
	maxDisplay := 5
	if len(allFindings) < maxDisplay {
		maxDisplay = len(allFindings)
	}

	for i := 0; i < maxDisplay; i++ {
		f := allFindings[i]
		emoji := "⚪"
		switch f.Severity {
		case findings.SeverityHigh:
			emoji = "🔴"
		case findings.SeverityMedium:
			emoji = "🟡"
		case findings.SeverityLow:
			emoji = "🟢"
		}

		fmt.Printf("%d. %s %s [%s]\n", i+1, emoji, f.Title, f.ID)
		fmt.Printf("   Files: %s\n", strings.Join(f.Files, ", "))
	}

	if len(allFindings) > maxDisplay {
		fmt.Printf("\n... and %d more finding(s)\n", len(allFindings)-maxDisplay)
	}
}

func createIssuesAuto(ctx context.Context, allFindings []findings.Finding) ([]int, error) {
	fmt.Println("\n📝 Creating GitHub issues...")

	// Get repo info
	owner, repo, err := getRepoInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo info: %w", err)
	}

	label := os.Getenv("AUTOENGINEER_LABEL")
	if label == "" {
		label = "autoengineer"
	}

	client, err := issues.NewClient(owner, repo, label)
	if err != nil {
		return nil, fmt.Errorf("failed to create issues client: %w", err)
	}

	// Ensure label exists
	if err := client.EnsureLabel(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to ensure label exists: %v\n", err)
	}

	created := 0
	skipped := 0
	failed := 0
	issueNums := []int{}

	for _, finding := range allFindings {
		// Check if issue exists
		if !flagForce {
			exists, matchType, err := client.IssueExists(ctx, finding.ID, finding.Title)
			if err != nil {
				fmt.Printf("⚠️  Warning: failed to check for existing issue: %v\n", err)
			}
			if exists {
				fmt.Printf("⏭️  Skipping (exists via %s): %s\n", matchType, finding.Title)
				skipped++
				continue
			}
		}

		fmt.Printf("📝 Creating: %s\n", finding.Title)
		issueNum, err := client.CreateIssue(ctx, finding)
		if err != nil {
			fmt.Printf("   ❌ Failed: %v\n", err)
			failed++
			continue
		}

		fmt.Printf("   ✅ Created [%s] #%d\n", finding.ID, issueNum)
		created++
		issueNums = append(issueNums, issueNum)
	}

	fmt.Println()
	fmt.Printf("📊 Summary: Created=%d, Skipped=%d, Failed=%d\n", created, skipped, failed)

	return issueNums, nil
}

func delegateIssues(ctx context.Context, issueNums []int) error {
	if len(issueNums) == 0 {
		return nil
	}

	fmt.Println("\n🤖 Delegating fixes to Copilot coding agent...")

	// Get repo info
	owner, repo, err := getRepoInfo()
	if err != nil {
		return fmt.Errorf("failed to get repo info: %w", err)
	}

	label := os.Getenv("AUTOENGINEER_LABEL")
	if label == "" {
		label = "autoengineer"
	}

	// Create issues client for adding delegated labels
	issuesClient, err := issues.NewClient(owner, repo, label)
	if err != nil {
		return fmt.Errorf("failed to create issues client: %w", err)
	}

	// Ensure delegated label exists
	if err := issuesClient.EnsureDelegatedLabel(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to ensure delegated label exists: %v\n", err)
	}

	client := copilot.NewClient()

	for _, issueNum := range issueNums {
		// Check if already delegated
		hasDelegated, err := issuesClient.HasDelegatedLabel(ctx, issueNum)
		if err != nil {
			fmt.Printf("   ⚠️  Warning: failed to check delegation status for issue #%d: %v\n", issueNum, err)
		} else if hasDelegated {
			fmt.Printf("⏭️  Skipping issue #%d (already delegated)\n", issueNum)
			continue
		}

		fmt.Printf("🔧 Delegating issue #%d...\n", issueNum)

		// Add delegated label
		if err := issuesClient.AddDelegatedLabel(ctx, issueNum); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to add delegated label to issue #%d: %v\n", issueNum, err)
		}

		prompt := fmt.Sprintf("Fix the issue described in %s/%s#%d", owner, repo, issueNum)

		if err := client.RunDelegate(ctx, prompt); err != nil {
			fmt.Printf("   ⚠️  Warning: delegation failed for issue #%d: %v\n", issueNum, err)
			continue
		}

		fmt.Printf("   ✅ Delegated issue #%d\n", issueNum)
	}

	fmt.Println()
	fmt.Printf("📊 Delegated %d issue(s) to Copilot coding agent\n", len(issueNums))

	return nil
}

func getRepoInfo() (owner, repo string, err error) {
	// Get remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	url := strings.TrimSpace(string(output))

	// Parse git@github.com:owner/repo.git or https://github.com/owner/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
	} else if strings.HasPrefix(url, "https://github.com/") {
		url = strings.TrimPrefix(url, "https://github.com/")
	}

	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid git remote URL")
	}

	return parts[0], parts[1], nil
}
