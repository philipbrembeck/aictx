//go:build darwin

package claudeauth

import (
	"fmt"
	"os/exec"
)

// Write writes Claude OAuth credentials to ALL macOS Keychain entries that use
// the Claude Code-credentials service, plus the .credentials.json file.
// There may be multiple entries (username-based, UUID-based) and we don't know
// which one Claude reads, so we update all of them.
func Write(credentials string) error {
	accounts := keychainAccounts()
	if len(accounts) == 0 {
		accounts = []string{keychainService}
	}

	var lastErr error
	for _, acct := range accounts {
		exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()

		cmd := exec.Command("security", "add-generic-password",
			"-s", keychainService,
			"-a", acct,
			"-w", credentials,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("keychain write failed for account %q: %w (%s)", acct, err, string(out))
		}
	}

	// Also write to .credentials.json.
	if err := writeToFile(credentials); err != nil {
		fmt.Printf("  ⚠ could not write .credentials.json: %v\n", err)
	}

	return lastErr
}
