package keyring

import (
	"testing"

	zalkeyring "github.com/zalando/go-keyring"
)

func TestKeyring_OAuth(t *testing.T) {
	zalkeyring.MockInit()

	const ctx = "work"
	const creds = `{"accessToken":"sk-ant-test","refreshToken":"rt-test","expiresAt":9999999999}`

	// Get on missing key returns error.
	_, err := GetOAuth(ctx)
	if err == nil {
		t.Error("GetOAuth() expected error when not set")
	}

	// Set then Get round-trip.
	if err := SetOAuth(ctx, creds); err != nil {
		t.Fatalf("SetOAuth() error: %v", err)
	}

	got, err := GetOAuth(ctx)
	if err != nil {
		t.Fatalf("GetOAuth() error: %v", err)
	}
	if got != creds {
		t.Errorf("GetOAuth() = %q, want %q", got, creds)
	}

	// Delete clears the entry.
	if err := DeleteOAuth(ctx); err != nil {
		t.Fatalf("DeleteOAuth() error: %v", err)
	}
	_, err = GetOAuth(ctx)
	if err == nil {
		t.Error("GetOAuth() expected error after delete")
	}

	// Delete on missing is no-op.
	if err := DeleteOAuth(ctx); err != nil {
		t.Errorf("DeleteOAuth() on missing error: %v", err)
	}
}

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
