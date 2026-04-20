//go:build darwin

package claudeauth

import (
	"fmt"
	"os/exec"
)

// Write writes Claude OAuth credentials to the macOS Keychain.
func Write(credentials string) error {
	// Delete existing entry first (add-generic-password -U can be unreliable).
	exec.Command("security", "delete-generic-password", "-s", keychainService).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", keychainService,
		"-a", keychainService,
		"-w", credentials,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write failed: %w (%s)", err, string(out))
	}
	return nil
}
