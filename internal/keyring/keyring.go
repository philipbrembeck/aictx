package keyring

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "aictx"

// account returns the keyring account name for a context+target combo.
func account(contextName, targetID string) string {
	return fmt.Sprintf("%s/%s", contextName, targetID)
}

// Set stores the API key for the given context and target in the OS keychain.
func Set(contextName, targetID, apiKey string) error {
	return keyring.Set(service, account(contextName, targetID), apiKey)
}

// Get retrieves the API key for the given context and target from the OS keychain.
func Get(contextName, targetID string) (string, error) {
	return keyring.Get(service, account(contextName, targetID))
}

// Delete removes the API key for the given context and target from the OS keychain.
// Returns nil if the entry does not exist.
func Delete(contextName, targetID string) error {
	err := keyring.Delete(service, account(contextName, targetID))
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
