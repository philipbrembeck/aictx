package claudeauth

import (
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound is returned when no Claude OAuth credentials are found.
var ErrNotFound = fmt.Errorf("no Claude OAuth credentials found")

// claudeDir returns the Claude config directory, respecting CLAUDE_CONFIG_DIR.
func claudeDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// credentialsFilePath returns the path to .credentials.json.
func credentialsFilePath() string {
	return filepath.Join(claudeDir(), ".credentials.json")
}

// Exists returns true if Claude OAuth credentials exist in any supported location.
func Exists() bool {
	_, err := Read()
	return err == nil
}

// readFromFile reads credentials from .credentials.json.
func readFromFile() (string, error) {
	data, err := os.ReadFile(credentialsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("reading Claude credentials: %w", err)
	}
	return string(data), nil
}

// writeToFile writes credentials to .credentials.json atomically with mode 0600.
func writeToFile(credentials string) error {
	path := credentialsFilePath()
	tmp := path + ".aictx-tmp"
	if err := os.WriteFile(tmp, []byte(credentials), 0600); err != nil {
		return fmt.Errorf("writing Claude credentials: %w", err)
	}
	return os.Rename(tmp, path)
}

// removeFile deletes .credentials.json. Returns nil if already absent.
func removeFile() error {
	err := os.Remove(credentialsFilePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
