//go:build darwin

package claudeauth

import "os/exec"

// Remove deletes Claude OAuth credentials from the macOS Keychain.
// Returns nil if entry doesn't exist.
func Remove() error {
	acct := keychainAccount()
	if acct == "" {
		return nil // no entry to remove
	}
	_ = exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()
	return nil
}
