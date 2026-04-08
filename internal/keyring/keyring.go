package keyring

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "aictx"

// Set stores the API key for the given context in the OS keychain.
func Set(contextName, apiKey string) error {
	return keyring.Set(service, contextName, apiKey)
}

// Get retrieves the API key for the given context from the OS keychain.
func Get(contextName string) (string, error) {
	return keyring.Get(service, contextName)
}

// Delete removes the API key for the given context from the OS keychain.
// Returns nil if the entry does not exist.
func Delete(contextName string) error {
	err := keyring.Delete(service, contextName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// GetLegacy reads an API key stored under the old "contextName/targetID" account format.
// Used only during one-time migration from the legacy per-target keyring scheme.
func GetLegacy(contextName, targetID string) (string, error) {
	return keyring.Get(service, fmt.Sprintf("%s/%s", contextName, targetID))
}

// DeleteLegacy removes an API key stored under the old "contextName/targetID" account format.
// Used only during one-time migration from the legacy per-target keyring scheme.
// Returns nil if the entry does not exist.
func DeleteLegacy(contextName, targetID string) error {
	err := keyring.Delete(service, fmt.Sprintf("%s/%s", contextName, targetID))
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
