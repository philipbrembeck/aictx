//go:build darwin

package claudeauth

import (
	"fmt"
	"os/exec"
)

// Write writes Claude OAuth credentials to the macOS Keychain.
// It discovers the account name Claude Code uses and replaces the same entry.
func Write(credentials string) error {
	acct := keychainAccount()
	if acct == "" {
		// No existing entry — use the service name as default account.
		acct = keychainService
	}

	// Delete existing entry using the discovered account.
	exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", acct,
		"-w", credentials,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write failed: %w (%s)", err, string(out))
	}
	return nil
}
