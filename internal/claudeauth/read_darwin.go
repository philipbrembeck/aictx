//go:build darwin

package claudeauth

import (
	"os/exec"
	"regexp"
	"strings"
)

const keychainService = "Claude Code-credentials"

// keychainAccount returns the account name ("acct" attribute) of the existing
// Keychain entry for keychainService. Claude Code picks its own account name,
// and we must reuse it for delete/add to hit the same entry.
func keychainAccount() string {
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	// Parse: "acct"<blob>="the-account-value"
	re := regexp.MustCompile(`"acct"<blob>="([^"]*)"`)
	m := re.FindSubmatch(out)
	if len(m) >= 2 {
		return string(m[1])
	}
	return ""
}

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
