package claudeauth

import (
	"encoding/json"
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

// claudeJSONPath returns the path to ~/.claude.json (account metadata).
func claudeJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude.json")
}

// ReadAccountMeta reads the oauthAccount JSON object from ~/.claude.json.
func ReadAccountMeta() (string, error) {
	data, err := os.ReadFile(claudeJSONPath())
	if err != nil {
		return "", fmt.Errorf("reading ~/.claude.json: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("parsing ~/.claude.json: %w", err)
	}

	acct, ok := raw["oauthAccount"]
	if !ok {
		return "", fmt.Errorf("no oauthAccount in ~/.claude.json")
	}
	return string(acct), nil
}

// WriteAccountMeta writes the oauthAccount JSON object into ~/.claude.json,
// preserving all other fields.
func WriteAccountMeta(accountJSON string) error {
	path := claudeJSONPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading ~/.claude.json: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing ~/.claude.json: %w", err)
	}

	raw["oauthAccount"] = json.RawMessage(accountJSON)

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling ~/.claude.json: %w", err)
	}
	out = append(out, '\n')

	tmp := path + ".aictx-tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return fmt.Errorf("writing ~/.claude.json: %w", err)
	}
	return os.Rename(tmp, path)
}
