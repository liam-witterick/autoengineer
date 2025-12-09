package findings

import "time"

// CodeSnippet represents a piece of code related to a finding
type CodeSnippet struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	Code      string `json:"code"`
}

// Finding represents a single issue discovered during analysis
type Finding struct {
	Category       string        `json:"category"`
	Title          string        `json:"title"`
	Severity       string        `json:"severity"`
	Description    string        `json:"description"`
	Recommendation string        `json:"recommendation"`
	Files          []string      `json:"files"`
	CodeSnippets   []CodeSnippet `json:"code_snippets,omitempty"`
}

// Severity levels
const (
	SeverityHigh   = "high"
	SeverityMedium = "medium"
	SeverityLow    = "low"
)

// Categories
const (
	CategorySecurity = "security"
	CategoryPipeline = "pipeline"
	CategoryInfra    = "infra"
)

// AcceptedFinding represents an accepted risk in the ignore config
type AcceptedFinding struct {
	Title        string    `yaml:"title"`
	Reason       string    `yaml:"reason,omitempty"`
	AcceptedBy   string    `yaml:"accepted_by,omitempty"`
	AcceptedDate time.Time `yaml:"accepted_date,omitempty"`
}

// SortBySeverity implements sort.Interface for []Finding based on severity
type BySeverity []Finding

func (a BySeverity) Len() int      { return len(a) }
func (a BySeverity) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySeverity) Less(i, j int) bool {
	severityOrder := map[string]int{
		SeverityHigh:   0,
		SeverityMedium: 1,
		SeverityLow:    2,
	}
	return severityOrder[a[i].Severity] < severityOrder[a[j].Severity]
}
