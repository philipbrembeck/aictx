package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IQNeoXen/aictx/internal/config"
	zalkeyring "github.com/zalando/go-keyring"
)

// captureStdout runs f, capturing everything written to os.Stdout.
func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// setupCopilotCmdEnv sets up a temporary home + config dir + mock keyring.
func setupCopilotCmdEnv(t *testing.T) {
	t.Helper()
	zalkeyring.MockInit()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}

// --- copilot status ---

func TestCopilotStatus_NotLoggedIn(t *testing.T) {
	setupCopilotCmdEnv(t)

	out := captureStdout(func() {
		copilotStatusRun(nil, nil) //nolint:errcheck
	})
	if !strings.Contains(out, "Not logged in") {
		t.Errorf("status output = %q, want 'Not logged in'", out)
	}
	if !strings.Contains(out, "copilot login") {
		t.Errorf("status output should mention 'copilot login'; got: %q", out)
	}
}

func TestCopilotStatus_LoggedIn(t *testing.T) {
	setupCopilotCmdEnv(t)

	// Simulate a login: set keyring + save config with login metadata and a Copilot context.
	zalkeyring.Set("aictx", "copilot-oauth", "gho_tok")

	loginTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	cfg := &config.Config{
		CopilotLogin: config.CopilotLogin{
			Username:   "testuser",
			LoggedInAt: loginTime,
		},
		Contexts: []config.Context{
			{
				Name:    "github-copilot",
				Provider: config.Provider{ProviderType: "copilot"},
				Targets: []config.TargetEntry{{ID: "pi-cli"}},
			},
		},
	}
	config.Save(cfg)

	out := captureStdout(func() {
		copilotStatusRun(nil, nil) //nolint:errcheck
	})
	if !strings.Contains(out, "testuser") {
		t.Errorf("status output missing username; got: %q", out)
	}
	if !strings.Contains(out, "github-copilot") {
		t.Errorf("status output missing context name; got: %q", out)
	}
	if !strings.Contains(out, "keychain") {
		t.Errorf("status output should mention keychain; got: %q", out)
	}
}

// --- copilot logout ---

func TestCopilotLogout_ClearsKeyring(t *testing.T) {
	setupCopilotCmdEnv(t)

	// Log in first.
	zalkeyring.Set("aictx", "copilot-oauth", "gho_tok")
	cfg := &config.Config{
		CopilotLogin: config.CopilotLogin{Username: "testuser"},
	}
	config.Save(cfg)

	// Run logout.
	if err := copilotLogoutRun(nil, nil); err != nil {
		t.Fatalf("copilotLogoutRun() error: %v", err)
	}

	// Keyring entry must be gone.
	_, err := zalkeyring.Get("aictx", "copilot-oauth")
	if err == nil {
		t.Error("copilot-oauth keyring entry still present after logout")
	}

	// Config must have cleared CopilotLogin.
	loaded, _ := config.Load()
	if loaded.CopilotLogin.Username != "" {
		t.Errorf("CopilotLogin.Username = %q after logout, want empty", loaded.CopilotLogin.Username)
	}
}

func TestCopilotLogout_NotLoggedIn(t *testing.T) {
	setupCopilotCmdEnv(t)

	out := captureStdout(func() {
		copilotLogoutRun(nil, nil) //nolint:errcheck
	})
	if !strings.Contains(out, "Not logged in") {
		t.Errorf("logout output = %q, want 'Not logged in'", out)
	}
}
