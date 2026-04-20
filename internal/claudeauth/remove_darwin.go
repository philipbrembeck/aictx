//go:build darwin

package claudeauth

import "os/exec"

// Remove deletes Claude OAuth credentials from both the macOS Keychain
// and .credentials.json.
func Remove() error {
	acct := keychainAccount()
	if acct != "" {
		_ = exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()
	}
	_ = removeFile()
	return nil
}
