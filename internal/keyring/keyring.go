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

const oauthAccountPrefix = "claude-oauth-"

// SetOAuth stores Claude OAuth credentials for the given context in the OS keychain.
func SetOAuth(contextName, credentials string) error {
	return keyring.Set(service, oauthAccountPrefix+contextName, credentials)
}

// GetOAuth retrieves Claude OAuth credentials for the given context from the OS keychain.
func GetOAuth(contextName string) (string, error) {
	return keyring.Get(service, oauthAccountPrefix+contextName)
}

// DeleteOAuth removes Claude OAuth credentials for the given context from the OS keychain.
// Returns nil if the entry does not exist.
func DeleteOAuth(contextName string) error {
	err := keyring.Delete(service, oauthAccountPrefix+contextName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

const copilotOAuthAccount = "copilot-oauth"

// SetCopilotOAuth stores the GitHub OAuth token used for Copilot in the OS keychain.
func SetCopilotOAuth(token string) error {
	return keyring.Set(service, copilotOAuthAccount, token)
}

// GetCopilotOAuth retrieves the GitHub OAuth token for Copilot from the OS keychain.
// Returns ("", keyring.ErrNotFound) if no token has been stored.
func GetCopilotOAuth() (string, error) {
	return keyring.Get(service, copilotOAuthAccount)
}

// DeleteCopilotOAuth removes the Copilot OAuth token from the OS keychain.
// Returns nil if the entry does not exist.
func DeleteCopilotOAuth() error {
	err := keyring.Delete(service, copilotOAuthAccount)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// IsCopilotLoggedIn returns true if a Copilot OAuth token is present in the keychain.
func IsCopilotLoggedIn() bool {
	_, err := GetCopilotOAuth()
	return err == nil
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
