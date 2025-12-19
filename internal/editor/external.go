package editor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Open launches an external editor with the given content.
// Returns the modified content after the editor exits.
func Open(content string) (string, error) {
	editor := getEditor()
	if editor == "" {
		return "", fmt.Errorf("no editor found: set $EDITOR environment variable")
	}

	// Create temp file with .json extension for syntax highlighting
	tmpFile, err := os.CreateTemp("", "avrocado-*.json")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write content to temp file
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	// Launch editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running editor: %w", err)
	}

	// Read modified content
	modified, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("reading modified file: %w", err)
	}

	return string(modified), nil
}

// getEditor returns the editor command to use.
// Checks $EDITOR, $VISUAL, then falls back to platform defaults.
func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// Platform-specific fallbacks
	switch runtime.GOOS {
	case "windows":
		return "notepad"
	default:
		// Try common editors
		for _, ed := range []string{"vim", "vi", "nano"} {
			if path, err := exec.LookPath(ed); err == nil {
				return path
			}
		}
	}

	return ""
}

// HasEditor checks if an external editor is available.
func HasEditor() bool {
	return getEditor() != ""
}
