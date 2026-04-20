//go:build darwin

package claudeauth

import (
	"os/exec"
	"strings"
)

const keychainService = "Claude Code-credentials"

// Read returns the current Claude OAuth credentials.
// On macOS, reads from the Keychain first; falls back to .credentials.json.
func Read() (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-w")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	return readFromFile()
}
