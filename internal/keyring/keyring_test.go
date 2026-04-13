package keyring

import (
	"testing"

	zalkeyring "github.com/zalando/go-keyring"
)

func TestKeyring_CopilotOAuth(t *testing.T) {
	zalkeyring.MockInit()

	// Initially not logged in.
	if IsCopilotLoggedIn() {
		t.Error("IsCopilotLoggedIn() = true before any Set, want false")
	}

	// Get on missing key returns an error.
	tok, err := GetCopilotOAuth()
	if err == nil {
		t.Errorf("GetCopilotOAuth() expected error when not set, got token %q", tok)
	}

	// Set then Get round-trip.
	const testToken = "gho_testtoken"
	if err := SetCopilotOAuth(testToken); err != nil {
		t.Fatalf("SetCopilotOAuth() error: %v", err)
	}

	if !IsCopilotLoggedIn() {
		t.Error("IsCopilotLoggedIn() = false after Set, want true")
	}

	got, err := GetCopilotOAuth()
	if err != nil {
		t.Fatalf("GetCopilotOAuth() error: %v", err)
	}
	if got != testToken {
		t.Errorf("GetCopilotOAuth() = %q, want %q", got, testToken)
	}

	// Delete clears the entry.
	if err := DeleteCopilotOAuth(); err != nil {
		t.Fatalf("DeleteCopilotOAuth() error: %v", err)
	}
	if IsCopilotLoggedIn() {
		t.Error("IsCopilotLoggedIn() = true after Delete, want false")
	}

	// Delete on already-missing entry is a no-op.
	if err := DeleteCopilotOAuth(); err != nil {
		t.Errorf("DeleteCopilotOAuth() on missing entry error: %v", err)
	}
}
