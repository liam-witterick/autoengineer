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
	"github.com/liam-witterick/autoengineer/go/internal/interactive"
	"github.com/liam-witterick/autoengineer/go/internal/issues"
	"github.com/liam-witterick/autoengineer/go/internal/progress"
	"github.com/liam-witterick/autoengineer/go/internal/scanner"
	"github.com/spf13/cobra"
)

const (
	version = "2.4.0"
)

var (
	flagAuto                 bool
	flagCreateIssues         bool
	flagDelegate             bool
	flagMinSeverity          string
	flagOutput               string
	flagForce                bool
	flagScope                string
	flagCheck                bool
	flagVersion              bool
	flagNoScanners           bool
	flagFast                 bool
	flagInstructions         string
	flagInstructionsText     string
	flagUseExistingFindings  bool
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
	rootCmd.Flags().StringVar(&flagInstructions, "instructions", "", "Path to custom instructions file")
	rootCmd.Flags().StringVar(&flagInstructionsText, "instructions-text", "", "Custom instructions as text")
	rootCmd.Flags().BoolVar(&flagUseExistingFindings, "use-existing-findings", false, "Load findings from file instead of running a new scan")

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

	// Load custom instructions
	var extraContext string
	
	// Priority order: --instructions-text > --instructions > .github/copilot-instructions.md
	if flagInstructionsText != "" {
		// Use inline instructions text
		extraContext = config.FormatInstructions(flagInstructionsText)
	} else if flagInstructions != "" {
		// Use custom instructions file
		instructions, err := config.LoadInstructions(flagInstructions)
		if err != nil {
			return fmt.Errorf("failed to load instructions from %s: %w", flagInstructions, err)
		}
		extraContext = instructions
	} else {
		// Try to load default instructions file
		instructions, err := config.LoadDefaultInstructions()
		if err != nil {
			return fmt.Errorf("failed to load default instructions: %w", err)
		}
		extraContext = instructions
	}

	// Check if scope is disabled
	if flagScope != "all" && cfg.IsScopeDisabled(flagScope) {
		fmt.Printf("⚠️  Scope '%s' is disabled in ignore config\n", flagScope)
		return nil
	}

	// Fetch existing tracked issues before analysis
	fmt.Println("\n🔍 Fetching existing tracked issues...")
	
	owner, repo, err := getRepoInfo()
	if err != nil {
		return fmt.Errorf("failed to get repo info: %w", err)
	}

	label := os.Getenv("AUTOENGINEER_LABEL")
	if label == "" {
		label = "autoengineer"
	}

	issuesClient, err := issues.NewClient(owner, repo, label)
	if err != nil {
		return fmt.Errorf("failed to create issues client: %w", err)
	}

	existingIssues, err := issuesClient.ListOpenIssues(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Warning: failed to fetch existing issues: %v\n", err)
		existingIssues = []issues.SearchResult{}
	} else {
		fmt.Printf("   Found %d existing tracked issue(s)\n", len(existingIssues))
	}

	var allFindings []findings.Finding
	var scannerStatuses []scanner.ScannerStatus

	// Load findings from file or run new scan
	if flagUseExistingFindings {
		// Load findings from existing file
		fmt.Printf("\n📂 Loading findings from %s...\n", flagOutput)
		
		loadedFindings, err := loadFindings(flagOutput)
		if err != nil {
			return err
		}
		
		fmt.Printf("   Loaded %d finding(s)\n", len(loadedFindings))
		allFindings = loadedFindings
	} else {
		// Build existing context for the analysis prompt
		existingContext := analysis.BuildExistingContext(existingIssues)

		// Run analysis with progress tracking
		fmt.Println("\n🔍 Running analysis...")
		fmt.Println()

		// Determine if scanners should run
		skipScanners := flagNoScanners || flagFast

		var err error
		allFindings, scannerStatuses, err = runAnalysisWithScanners(ctx, flagScope, cfg, scannerCfg, skipScanners, existingContext, extraContext)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		fmt.Println()

		// Display scanner summary
		if !skipScanners && len(scannerStatuses) > 0 {
			displayScannerSummary(scannerStatuses)
		}

		// Run intelligent deduplication with Copilot
		// This merges related findings across categories and filters out duplicates of existing issues
		if len(allFindings) > 0 {
			fmt.Println()
			fmt.Println("🔄 Deduplicating findings...")
			
			client := copilot.NewClient()
			deduplicated, err := client.RunDeduplication(ctx, allFindings, existingIssues)
			if err != nil {
				// Log the error but continue with original findings
				fmt.Printf("   ⚠️  Warning: deduplication failed, continuing with original findings: %v\n", err)
			} else {
				beforeDedup := len(allFindings)
				allFindings = deduplicated
				deduplicatedCount := beforeDedup - len(allFindings)
				if deduplicatedCount > 0 {
					fmt.Printf("   Removed %d duplicate/related finding(s)\n", deduplicatedCount)
				} else {
					fmt.Printf("   No duplicates found\n")
				}
			}
		}
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

	// Save findings to file (only when running new scan)
	// We skip saving when using existing findings to avoid overwriting
	// the original file with potentially filtered/modified results
	if !flagUseExistingFindings {
		if err := saveFindings(filtered, flagOutput); err != nil {
			return fmt.Errorf("failed to save findings: %w", err)
		}
	}

	// Display preview with existing issues
	displayPreview(existingIssues, filtered, ignoredCount)

	if len(filtered) == 0 && len(existingIssues) == 0 {
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

	// Interactive mode (owner, repo, label already fetched above)
	session, err := interactive.NewSession(filtered, owner, repo, label)
	if err != nil {
		return fmt.Errorf("failed to create interactive session: %w", err)
	}

	return session.Run(ctx)
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

	// Check for custom instructions
	fmt.Println()
	fmt.Println("🔍 Custom Instructions:")
	if config.CheckInstructionsExists() {
		fmt.Println("   ✅ .github/copilot-instructions.md (found)")
	} else {
		fmt.Println("   ⏭️  .github/copilot-instructions.md (not found - will be skipped)")
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

func runAnalysis(ctx context.Context, scope string, cfg *config.IgnoreConfig, tracker *progress.ScopeTracker, existingContext string, extraContext string) ([]findings.Finding, error) {
	client := copilot.NewClient()

	base := analysis.BaseAnalyzer{
		Client:          client,
		ExistingContext: existingContext,
		ExtraContext:    extraContext,
	}

	var allFindings []findings.Finding

	switch scope {
	case "security":
		if cfg.IsScopeDisabled("security") {
			fmt.Println("   ⚠️  Security scope is disabled")
			return []findings.Finding{}, nil
		}
		if tracker != nil {
			tracker.StartScope("security")
		}
		analyzer := analysis.NewSecurityAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			if tracker != nil {
				tracker.FailScope("security", err)
			}
			return nil, err
		}
		if tracker != nil {
			tracker.CompleteScope("security", len(results))
		}
		allFindings = results

	case "pipeline":
		if cfg.IsScopeDisabled("pipeline") {
			fmt.Println("   ⚠️  Pipeline scope is disabled")
			return []findings.Finding{}, nil
		}
		if tracker != nil {
			tracker.StartScope("pipeline")
		}
		analyzer := analysis.NewPipelineAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			if tracker != nil {
				tracker.FailScope("pipeline", err)
			}
			return nil, err
		}
		if tracker != nil {
			tracker.CompleteScope("pipeline", len(results))
		}
		allFindings = results

	case "infra":
		if cfg.IsScopeDisabled("infra") {
			fmt.Println("   ⚠️  Infra scope is disabled")
			return []findings.Finding{}, nil
		}
		if tracker != nil {
			tracker.StartScope("infra")
		}
		analyzer := analysis.NewInfraAnalyzer(base)
		results, err := analyzer.Run(ctx)
		if err != nil {
			if tracker != nil {
				tracker.FailScope("infra", err)
			}
			return nil, err
		}
		if tracker != nil {
			tracker.CompleteScope("infra", len(results))
		}
		allFindings = results

	case "all":
		// Run all scopes concurrently
		type result struct {
			scope    string
			findings []findings.Finding
			err      error
		}

		scopes := []string{"security", "pipeline", "infra"}
		enabledScopes := []string{}

		for _, s := range scopes {
			if !cfg.IsScopeDisabled(s) {
				enabledScopes = append(enabledScopes, s)
			}
		}

		if len(enabledScopes) == 0 {
			return []findings.Finding{}, nil
		}

		resultCh := make(chan result, len(enabledScopes))

		// Start each enabled scope
		for _, s := range enabledScopes {
			if tracker != nil {
				tracker.StartScope(s)
			}

			go func(scopeName string) {
				// Create a new client for each concurrent scope to avoid race conditions
				scopeClient := copilot.NewClient()
				scopeBase := analysis.BaseAnalyzer{
					Client:          scopeClient,
					ExistingContext: existingContext,
					ExtraContext:    extraContext,
				}

				var analyzer analysis.Analyzer
				switch scopeName {
				case "security":
					analyzer = analysis.NewSecurityAnalyzer(scopeBase)
				case "pipeline":
					analyzer = analysis.NewPipelineAnalyzer(scopeBase)
				case "infra":
					analyzer = analysis.NewInfraAnalyzer(scopeBase)
				}

				results, err := analyzer.Run(ctx)
				resultCh <- result{scope: scopeName, findings: results, err: err}
			}(s)
		}

		// Collect results
		allResults := make([]result, 0, len(enabledScopes))
		for i := 0; i < len(enabledScopes); i++ {
			res := <-resultCh
			allResults = append(allResults, res)

			if tracker != nil {
				if res.err != nil {
					tracker.FailScope(res.scope, res.err)
				} else {
					tracker.CompleteScope(res.scope, len(res.findings))
				}
			}
		}

		// Check for errors and merge findings with deduplication
		var findingSlices [][]findings.Finding
		for _, res := range allResults {
			if res.err != nil {
				return nil, fmt.Errorf("%s analysis failed: %w", res.scope, res.err)
			}
			findingSlices = append(findingSlices, res.findings)
		}
		allFindings = findings.Merge(findingSlices...)

	default:
		return nil, fmt.Errorf("invalid scope: %s (must be security|pipeline|infra|all)", scope)
	}

	return allFindings, nil
}

// runAnalysisWithScanners runs both Copilot analysis and external scanners in parallel
func runAnalysisWithScanners(ctx context.Context, scope string, cfg *config.IgnoreConfig, scannerCfg *config.ScannerConfig, skipScanners bool, existingContext string, extraContext string) ([]findings.Finding, []scanner.ScannerStatus, error) {
	type result struct {
		findings []findings.Finding
		statuses []scanner.ScannerStatus
		err      error
	}

	// Determine scopes to analyze
	var scopes []string
	if scope == "all" {
		scopes = []string{"security", "pipeline", "infra"}
		// Filter out disabled scopes
		enabledScopes := []string{}
		for _, s := range scopes {
			if !cfg.IsScopeDisabled(s) {
				enabledScopes = append(enabledScopes, s)
			}
		}
		scopes = enabledScopes
	} else {
		if !cfg.IsScopeDisabled(scope) {
			scopes = []string{scope}
		} else {
			scopes = []string{}
		}
	}

	// Create progress tracker
	var tracker *progress.ScopeTracker
	if len(scopes) > 0 {
		tracker = progress.NewScopeTracker(scopes)
	}

	// Run Copilot analysis and scanners in parallel
	copilotCh := make(chan result, 1)
	scannerCh := make(chan result, 1)

	// Run Copilot analysis
	go func() {
		copilotFindings, err := runAnalysis(ctx, scope, cfg, tracker, existingContext, extraContext)
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

	// Cleanup progress tracker
	if tracker != nil {
		tracker.Finish()
	}

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

func loadFindings(inputPath string) ([]findings.Finding, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("findings file not found: %s", inputPath)
		}
		return nil, fmt.Errorf("failed to read findings file: %w", err)
	}

	var allFindings []findings.Finding
	if err := json.Unmarshal(data, &allFindings); err != nil {
		return nil, fmt.Errorf("failed to parse findings file: %w", err)
	}

	return allFindings, nil
}

func displayPreview(existingIssues []issues.SearchResult, allFindings []findings.Finding, ignoredCount int) {
	fmt.Println()
	
	// Display existing tracked issues
	if len(existingIssues) > 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("📋 TRACKED ISSUES (existing GitHub issues)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		
		for i, issue := range existingIssues {
			fmt.Printf("%d. [#%d] %s\n", i+1, issue.Number, issue.Title)
		}
		
		fmt.Println()
	}
	
	// Display new findings
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 NEW FINDINGS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	findings.DisplaySummary(allFindings)

	if ignoredCount > 0 {
		fmt.Printf("\n⏭️  Ignored:        %d finding(s) (based on ignore config)\n", ignoredCount)
	}

	fmt.Println()

	findings.DisplayFindings(allFindings, findings.DefaultDisplayOptions())
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
			exists, matchType, err := client.IssueExists(ctx, finding.Title)
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

		fmt.Printf("   ✅ Created #%d\n", issueNum)
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

		// Assign copilot as assignee to trigger Copilot coding agent
		if err := issuesClient.AssignCopilot(ctx, issueNum); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to assign copilot to issue #%d: %v\n", issueNum, err)
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
