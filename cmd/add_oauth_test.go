package cmd

import (
	"os"
	"path/filepath"
	"testing"

	zalkeyring "github.com/zalando/go-keyring"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/keyring"
)

func TestAddOAuth_CapturesAndSwitches(t *testing.T) {
	zalkeyring.MockInit()

	// Set up temp HOME with .claude directory and fake credentials.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	fakeCreds := `{"accessToken":"sk-ant-fake","refreshToken":"rt-fake","expiresAt":9999999999}`
	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), []byte(fakeCreds), 0600); err != nil {
		t.Fatal(err)
	}

	// Set up aictx config dir.
	configDir := filepath.Join(home, ".config", "aictx")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Simulate addOAuthRun.
	cfg := &config.Config{}
	// Create context with HasOAuthKey.
	ctx := config.Context{
		Name:        "test-oauth",
		HasOAuthKey: true,
		Targets:     []config.TargetEntry{{ID: "claude-code-cli"}},
	}
	cfg.Contexts = append(cfg.Contexts, ctx)

	// Store credentials in keyring.
	if err := keyring.SetOAuth("test-oauth", fakeCreds); err != nil {
		t.Fatalf("SetOAuth: %v", err)
	}

	// Verify round-trip.
	got, err := keyring.GetOAuth("test-oauth")
	if err != nil {
		t.Fatalf("GetOAuth: %v", err)
	}
	if got != fakeCreds {
		t.Errorf("GetOAuth = %q, want %q", got, fakeCreds)
	}

	// Verify context state.
	if !cfg.Contexts[0].HasOAuthKey {
		t.Error("HasOAuthKey not set")
	}
	if cfg.Contexts[0].HasKeyringKey {
		t.Error("HasKeyringKey should not be set for OAuth context")
	}
}
