//go:build darwin

package claudeauth

import (
	"os/exec"
	"regexp"
	"strings"
)

const keychainService = "Claude Code-credentials"

// keychainAccounts returns all account names for the Claude Code-credentials
// service. There may be multiple entries (e.g. username-based and UUID-based).
func keychainAccounts() []string {
	cmd := exec.Command("security", "dump-keychain")
	out, _ := cmd.Output()

	lines := strings.Split(string(out), "\n")
	re := regexp.MustCompile(`"acct"<blob>="([^"]*)"`)
	svcRe := regexp.MustCompile(`"svce"<blob>="` + regexp.QuoteMeta(keychainService) + `"`)

	var accounts []string
	for i, line := range lines {
		if svcRe.MatchString(line) {
			// Look backwards for the acct line in this entry block
			for j := i - 1; j >= 0 && j >= i-15; j-- {
				if m := re.FindStringSubmatch(lines[j]); m != nil {
					accounts = append(accounts, m[1])
					break
				}
			}
		}
	}
	return accounts
}

// Read returns the current Claude OAuth credentials.
// On macOS, reads from the Keychain first; falls back to .credentials.json.
func Read() (string, error) {
	// Try each known account for this service.
	for _, acct := range keychainAccounts() {
		cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", acct, "-w")
		out, err := cmd.Output()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" {
				return val, nil
			}
		}
	}
	// Fallback: try without specifying account (returns first match).
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-w")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	return readFromFile()
}
