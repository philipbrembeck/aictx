package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/copilot"
	zalkeyring "github.com/zalando/go-keyring"
)

// setupCopilotEnv sets up a temporary HOME/.pi/agent dir and mocks the keyring.
// Returns the config loaded from a temporary XDG config dir.
func setupCopilotEnv(t *testing.T) {
	t.Helper()
	zalkeyring.MockInit()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	// Create .pi/agent dir so pi-cli Detect() returns true.
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0755); err != nil {
		t.Fatalf("MkdirAll .pi/agent: %v", err)
	}
}

func TestSwitchContext_CopilotProvider_FetchesToken(t *testing.T) {
	setupCopilotEnv(t)

	// Mock Copilot token exchange endpoint.
	expiresAt := time.Now().Add(30 * time.Minute).Unix()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      "tid=freshcopilottoken",
			"expires_at": expiresAt,
		})
	}))
	defer tokenSrv.Close()

	origURL := copilot.CopilotTokenURL
	copilot.CopilotTokenURL = tokenSrv.URL
	t.Cleanup(func() { copilot.CopilotTokenURL = origURL })

	// Store an OAuth token in the mock keyring.
	if err := zalkeyring.Set("aictx", "copilot-oauth", "gho_oauthtoken"); err != nil {
		t.Fatalf("keyring.Set: %v", err)
	}

	// Build a config with a Copilot context.
	cfg := &config.Config{
		Contexts: []config.Context{
			{
				Name: "github-copilot",
				Provider: config.Provider{
					Endpoint:     copilot.CopilotAPIEndpoint,
					Model:        "gpt-4o",
					ProviderType: "copilot",
				},
				Targets: []config.TargetEntry{{ID: "pi-cli"}},
			},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	if err := switchContext(reloaded, "github-copilot"); err != nil {
		t.Fatalf("switchContext() error: %v", err)
	}

	// The extension file should contain the fresh Copilot token.
	home := os.Getenv("HOME")
	extPath := filepath.Join(home, ".pi", "agent", "extensions", "aictx-provider.ts")
	data, err := os.ReadFile(extPath)
	if err != nil {
		t.Fatalf("reading extension file: %v", err)
	}
	ext := string(data)
	if !strings.Contains(ext, "tid=freshcopilottoken") {
		t.Errorf("extension does not contain fresh Copilot token; got:\n%s", ext)
	}
	if !strings.Contains(ext, `"copilot"`) {
		t.Errorf("extension missing \"copilot\" provider name; got:\n%s", ext)
	}
	if !strings.Contains(ext, `"openai-completions"`) {
		t.Errorf("extension missing openai-completions api; got:\n%s", ext)
	}
}

func TestSwitchContext_CopilotProvider_SkipsClaudeTargets(t *testing.T) {
	setupCopilotEnv(t)

	expiresAt := time.Now().Add(30 * time.Minute).Unix()
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      "tid=tok",
			"expires_at": expiresAt,
		})
	}))
	defer tokenSrv.Close()

	origURL := copilot.CopilotTokenURL
	copilot.CopilotTokenURL = tokenSrv.URL
	t.Cleanup(func() { copilot.CopilotTokenURL = origURL })

	zalkeyring.Set("aictx", "copilot-oauth", "gho_oauthtoken")

	cfg := &config.Config{
		Contexts: []config.Context{
			{
				Name: "github-copilot",
				Provider: config.Provider{
					Endpoint:     copilot.CopilotAPIEndpoint,
					Model:        "gpt-4o",
					ProviderType: "copilot",
				},
				// Both pi-cli and claude-code-cli listed, but only pi-cli should be applied.
				Targets: []config.TargetEntry{
					{ID: "pi-cli"},
					{ID: "claude-code-cli"},
				},
			},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}
	reloaded, _ := config.Load()

	// Capture stderr to verify warning is emitted for claude-code-cli.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := switchContext(reloaded, "github-copilot")

	w.Close()
	os.Stderr = origStderr
	var stderrBuf strings.Builder
	buf := make([]byte, 4096)
	for {
		n, _ := r.Read(buf)
		if n == 0 {
			break
		}
		stderrBuf.Write(buf[:n])
	}

	if err != nil {
		t.Fatalf("switchContext() error: %v", err)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "Claude Code CLI") || !strings.Contains(stderr, "skipped") {
		t.Errorf("expected warning about Claude Code CLI being skipped; stderr:\n%s", stderr)
	}

	// Extension file must exist (pi-cli was applied).
	home := os.Getenv("HOME")
	extPath := filepath.Join(home, ".pi", "agent", "extensions", "aictx-provider.ts")
	if _, err := os.Stat(extPath); err != nil {
		t.Errorf("extension file missing — pi-cli was not applied: %v", err)
	}
}

func TestSwitchContext_CopilotProvider_NotLoggedIn(t *testing.T) {
	setupCopilotEnv(t)
	// No OAuth token stored — keyring is empty.

	cfg := &config.Config{
		Contexts: []config.Context{
			{
				Name: "github-copilot",
				Provider: config.Provider{
					Endpoint:     copilot.CopilotAPIEndpoint,
					Model:        "gpt-4o",
					ProviderType: "copilot",
				},
				Targets: []config.TargetEntry{{ID: "pi-cli"}},
			},
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save: %v", err)
	}
	reloaded, _ := config.Load()

	err := switchContext(reloaded, "github-copilot")
	if err == nil {
		t.Fatal("switchContext() expected error when not logged in, got nil")
	}
	if !strings.Contains(err.Error(), "not logged in") && !strings.Contains(err.Error(), "copilot login") {
		t.Errorf("error = %q, want helpful message about 'copilot login'", err.Error())
	}
}
