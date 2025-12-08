package config

import (
	"fmt"
	"os"
)

// FormatInstructions wraps instructions text with clear formatting
func FormatInstructions(text string) string {
	if text == "" {
		return ""
	}
	return fmt.Sprintf("\n\nCUSTOM INSTRUCTIONS FROM USER:\n%s\nEND CUSTOM INSTRUCTIONS\n", text)
}

// LoadInstructions loads custom instructions from a file or returns empty string if not found
// The function checks for the file existence and reads it, wrapping the content in a clear format
func LoadInstructions(path string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // File doesn't exist, return empty string (not an error)
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read instructions file: %w", err)
	}

	// If file is empty, return empty string
	if len(data) == 0 {
		return "", nil
	}

	// Format the instructions with clear wrapping
	return FormatInstructions(string(data)), nil
}

// LoadDefaultInstructions attempts to load the default .github/copilot-instructions.md file
func LoadDefaultInstructions() (string, error) {
	return LoadInstructions(".github/copilot-instructions.md")
}

// CheckInstructionsExists checks if the default instructions file exists
func CheckInstructionsExists() bool {
	_, err := os.Stat(".github/copilot-instructions.md")
	return err == nil
}
