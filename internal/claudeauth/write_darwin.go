//go:build darwin

package claudeauth

import (
	"fmt"
	"os/exec"
)

// Write writes Claude OAuth credentials to both the macOS Keychain and
// the .credentials.json file, covering whichever location Claude reads from.
func Write(credentials string) error {
	// Write to Keychain.
	acct := keychainAccount()
	if acct == "" {
		acct = keychainService
	}
	exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", acct).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", acct,
		"-w", credentials,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write failed: %w (%s)", err, string(out))
	}

	// Also write to .credentials.json as fallback.
	if err := writeToFile(credentials); err != nil {
		fmt.Printf("  ⚠ could not write .credentials.json: %v\n", err)
	}

	return nil
}
