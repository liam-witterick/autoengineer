package findings

import (
	"fmt"
	"strings"
)

// DisplayOptions controls how findings are displayed
type DisplayOptions struct {
	ShowCategory     bool
	ShowDescription  bool
	ShowCodeSnippets bool
	TruncateDesc     int
	ShowAll          bool
	MaxDisplay       int
}

// DefaultDisplayOptions returns default display settings
func DefaultDisplayOptions() DisplayOptions {
	return DisplayOptions{
		ShowCategory:     false,
		ShowDescription:  false,
		ShowCodeSnippets: false,
		TruncateDesc:     0,
		ShowAll:          false,
		MaxDisplay:       5,
	}
}

// DetailedDisplayOptions returns settings for detailed display
func DetailedDisplayOptions() DisplayOptions {
	return DisplayOptions{
		ShowCategory:     true,
		ShowDescription:  true,
		ShowCodeSnippets: true,
		TruncateDesc:     150,
		ShowAll:          true,
		MaxDisplay:       0,
	}
}

// DisplayFindings displays a list of findings with the given options
func DisplayFindings(findings []Finding, opts DisplayOptions) {
	maxDisplay := opts.MaxDisplay
	if opts.ShowAll || len(findings) < maxDisplay {
		maxDisplay = len(findings)
	}

	for i := 0; i < maxDisplay; i++ {
		f := findings[i]
		emoji := SeverityEmoji(f.Severity)
		fmt.Printf("%d. %s %s\n", i+1, emoji, f.Title)
		
		if opts.ShowCategory {
			fmt.Printf("   Category: %s\n", f.Category)
		}
		
		fmt.Printf("   Files: %s\n", joinFiles(f.Files))
		
		if opts.ShowDescription && f.Description != "" {
			desc := f.Description
			if opts.TruncateDesc > 0 && len(desc) > opts.TruncateDesc {
				desc = desc[:opts.TruncateDesc] + "..."
			}
			fmt.Printf("   Description: %s\n", desc)
		}
		
		if opts.ShowCodeSnippets && len(f.CodeSnippets) > 0 {
			displayCodeSnippets(f.CodeSnippets)
		}
		
		if opts.ShowAll {
			fmt.Println()
		}
	}

	if !opts.ShowAll && len(findings) > maxDisplay {
		fmt.Printf("\n... and %d more finding(s)\n", len(findings)-maxDisplay)
	}
}

// DisplaySummary displays a summary of findings by severity and category
func DisplaySummary(findings []Finding) {
	high, medium, low := CountBySeverity(findings)
	security, pipeline, infra := CountByCategory(findings)
	total := len(findings)

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
}

// SeverityEmoji returns the emoji for a severity level
func SeverityEmoji(severity string) string {
	switch severity {
	case SeverityHigh:
		return "🔴"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🟢"
	default:
		return "⚪"
	}
}

// joinFiles joins file paths with commas
func joinFiles(files []string) string {
	return strings.Join(files, ", ")
}

// displayCodeSnippets displays code snippets with proper formatting
func displayCodeSnippets(snippets []CodeSnippet) {
	for _, snippet := range snippets {
		// Format the header with file and line numbers
		header := fmt.Sprintf("   Code (%s", snippet.File)
		if snippet.StartLine > 0 {
			if snippet.EndLine > 0 && snippet.EndLine != snippet.StartLine {
				header += fmt.Sprintf(":%d-%d", snippet.StartLine, snippet.EndLine)
			} else {
				header += fmt.Sprintf(":%d", snippet.StartLine)
			}
		}
		header += "):"
		fmt.Println(header)
		
		// Display the code with indentation
		fmt.Println("   ```")
		lines := strings.Split(snippet.Code, "\n")
		for _, line := range lines {
			fmt.Printf("   %s\n", line)
		}
		fmt.Println("   ```")
	}
}
