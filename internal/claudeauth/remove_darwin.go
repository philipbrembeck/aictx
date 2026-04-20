//go:build darwin

package claudeauth

import "os/exec"

// Remove deletes all Claude OAuth credentials Keychain entries and .credentials.json.
func Remove() error {
	for _, acct := range keychainAccounts() {
		_ = exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()
	}
	_ = removeFile()
	return nil
}
