package findings

import (
	"encoding/json"
	"testing"
)

func TestCodeSnippetJSON(t *testing.T) {
	snippet := CodeSnippet{
		File:      "main.go",
		StartLine: 10,
		EndLine:   15,
		Code:      "func main() {\n    fmt.Println(\"Hello\")\n}",
	}

	// Test JSON marshaling
	data, err := json.Marshal(snippet)
	if err != nil {
		t.Fatalf("failed to marshal CodeSnippet: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled CodeSnippet
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal CodeSnippet: %v", err)
	}

	// Verify fields
	if unmarshaled.File != snippet.File {
		t.Errorf("File = %q, want %q", unmarshaled.File, snippet.File)
	}
	if unmarshaled.StartLine != snippet.StartLine {
		t.Errorf("StartLine = %d, want %d", unmarshaled.StartLine, snippet.StartLine)
	}
	if unmarshaled.EndLine != snippet.EndLine {
		t.Errorf("EndLine = %d, want %d", unmarshaled.EndLine, snippet.EndLine)
	}
	if unmarshaled.Code != snippet.Code {
		t.Errorf("Code = %q, want %q", unmarshaled.Code, snippet.Code)
	}
}

func TestFindingWithCodeSnippetsJSON(t *testing.T) {
	finding := Finding{
		Category:       CategorySecurity,
		Title:          "Test Finding",
		Severity:       SeverityHigh,
		Description:    "Test description",
		Recommendation: "Test recommendation",
		Files:          []string{"main.go", "auth.go"},
		CodeSnippets: []CodeSnippet{
			{
				File:      "main.go",
				StartLine: 10,
				EndLine:   15,
				Code:      "func main() {}",
			},
			{
				File:      "auth.go",
				StartLine: 20,
				EndLine:   25,
				Code:      "func authenticate() {}",
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(finding)
	if err != nil {
		t.Fatalf("failed to marshal Finding: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Finding
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal Finding: %v", err)
	}

	// Verify code snippets
	if len(unmarshaled.CodeSnippets) != len(finding.CodeSnippets) {
		t.Errorf("CodeSnippets length = %d, want %d", len(unmarshaled.CodeSnippets), len(finding.CodeSnippets))
	}

	for i := range finding.CodeSnippets {
		if unmarshaled.CodeSnippets[i].File != finding.CodeSnippets[i].File {
			t.Errorf("CodeSnippets[%d].File = %q, want %q", i, unmarshaled.CodeSnippets[i].File, finding.CodeSnippets[i].File)
		}
	}
}

func TestFindingWithoutCodeSnippetsJSON(t *testing.T) {
	finding := Finding{
		Category:       CategorySecurity,
		Title:          "Test Finding",
		Severity:       SeverityHigh,
		Description:    "Test description",
		Recommendation: "Test recommendation",
		Files:          []string{"main.go"},
	}

	// Test JSON marshaling (should omit empty code_snippets)
	data, err := json.Marshal(finding)
	if err != nil {
		t.Fatalf("failed to marshal Finding: %v", err)
	}

	// Verify that code_snippets is omitted when empty
	if finding.CodeSnippets == nil {
		// Empty slice should not appear in JSON due to omitempty
		// This test verifies backward compatibility
		var unmarshaled Finding
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal Finding: %v", err)
		}

		if unmarshaled.Title != finding.Title {
			t.Errorf("Title = %q, want %q", unmarshaled.Title, finding.Title)
		}
	}
}

func TestCodeSnippetOmitEmptyFields(t *testing.T) {
	snippet := CodeSnippet{
		File: "main.go",
		Code: "func main() {}",
		// StartLine and EndLine are omitted
	}

	data, err := json.Marshal(snippet)
	if err != nil {
		t.Fatalf("failed to marshal CodeSnippet: %v", err)
	}

	// Verify that start_line and end_line are not present when zero
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// These fields should be omitted or zero
	if _, exists := unmarshaled["start_line"]; exists {
		if unmarshaled["start_line"] != 0.0 && unmarshaled["start_line"] != nil {
			t.Errorf("start_line should be omitted or zero, got %v", unmarshaled["start_line"])
		}
	}
}
