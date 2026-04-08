package claudecli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/IQNeoXen/aictx/internal/config"
)

// setupClaude creates a temp HOME with the .claude directory and returns a fresh Target.
func setupClaude(t *testing.T) *Target {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0755); err != nil {
		t.Fatalf("MkdirAll .claude: %v", err)
	}
	return New()
}

func writeSettings(t *testing.T, tgt *Target, content string) {
	t.Helper()
	if err := os.WriteFile(tgt.settingsPath(), []byte(content), 0644); err != nil {
		t.Fatalf("writeSettings: %v", err)
	}
}

func readSettingsMap(t *testing.T, tgt *Target) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(tgt.settingsPath())
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	return m
}

func envMap(t *testing.T, m map[string]interface{}) map[string]string {
	t.Helper()
	raw, ok := m["env"].(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("env[%s] is not a string: %T", k, v)
		}
		out[k] = s
	}
	return out
}

// ---------- Detect ----------

func TestDetect_NotInstalled(t *testing.T) {
	tgt := setupClaude(t)
	// Remove the .claude dir entirely so neither file exists.
	home, _ := os.UserHomeDir()
	_ = os.RemoveAll(filepath.Join(home, ".claude"))
	if tgt.Detect() {
		t.Error("Detect() = true, want false when no files present")
	}
}

func TestDetect_SettingsExists(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)
	if !tgt.Detect() {
		t.Error("Detect() = false, want true when settings.json exists")
	}
}

// ---------- Apply ----------

func TestApply_BasicProvider(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)

	te := config.TargetEntry{
		Provider: config.Provider{
			Endpoint: "https://api.example.com",
			APIKey:   "sk-test",
			Model:    "claude-3",
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	env := envMap(t, readSettingsMap(t, tgt))
	if env["ANTHROPIC_BASE_URL"] != "https://api.example.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %q", env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "sk-test" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %q", env["ANTHROPIC_AUTH_TOKEN"])
	}
	if env["ANTHROPIC_MODEL"] != "claude-3" {
		t.Errorf("ANTHROPIC_MODEL = %q", env["ANTHROPIC_MODEL"])
	}
}

func TestApply_MergesExistingSettings(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{"someOtherKey": true}`)

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "m"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if m["someOtherKey"] != true {
		t.Error("someOtherKey was lost after Apply()")
	}
}

func TestApply_MergesExistingEnvKeys(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{"env": {"EXISTING_KEY": "existing-value"}}`)

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "new-model"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	env := envMap(t, readSettingsMap(t, tgt))
	if env["EXISTING_KEY"] != "existing-value" {
		t.Error("existing env key was removed when it should have been preserved")
	}
	if env["ANTHROPIC_MODEL"] != "new-model" {
		t.Errorf("ANTHROPIC_MODEL = %q", env["ANTHROPIC_MODEL"])
	}
}

func TestApply_CleansUpPrevEnvKeys(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{"env": {"ANTHROPIC_MODEL": "old-model", "ANTHROPIC_AUTH_TOKEN": "old-key"}}`)
	tgt.PrevEnvKeys = []string{"ANTHROPIC_MODEL", "ANTHROPIC_AUTH_TOKEN"}

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "new-model"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	env := envMap(t, readSettingsMap(t, tgt))
	if _, ok := env["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Error("ANTHROPIC_AUTH_TOKEN should have been removed (in PrevEnvKeys)")
	}
	if env["ANTHROPIC_MODEL"] != "new-model" {
		t.Errorf("ANTHROPIC_MODEL = %q, want new-model", env["ANTHROPIC_MODEL"])
	}
}

func TestApply_TrailingCommaHandled(t *testing.T) {
	tgt := setupClaude(t)
	// VSCode-style JSON with trailing comma.
	writeSettings(t, tgt, `{"existingKey": "value",}`)

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "m"}}); err != nil {
		t.Fatalf("Apply() with trailing comma: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if m["existingKey"] != "value" {
		t.Error("existingKey lost after handling trailing comma")
	}
}

func TestApply_AlwaysThinking(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)

	b := true
	te := config.TargetEntry{
		Options: config.Options{AlwaysThinking: &b},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if m["alwaysThinkingEnabled"] != true {
		t.Errorf("alwaysThinkingEnabled = %v, want true", m["alwaysThinkingEnabled"])
	}
}

func TestApply_Headers(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)

	te := config.TargetEntry{
		Provider: config.Provider{
			Headers: map[string]string{"X-Proxy-Auth": "token123"},
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	env := envMap(t, readSettingsMap(t, tgt))
	raw := env["ANTHROPIC_CUSTOM_HEADERS"]
	if raw == "" {
		t.Fatal("ANTHROPIC_CUSTOM_HEADERS not set")
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		t.Fatalf("unmarshal headers: %v", err)
	}
	if headers["X-Proxy-Auth"] != "token123" {
		t.Errorf("X-Proxy-Auth = %q", headers["X-Proxy-Auth"])
	}
}

func TestApply_EmptyProvider(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)

	if err := tgt.Apply(config.TargetEntry{}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if _, ok := m["env"]; ok {
		t.Error("env key should be absent when provider is empty")
	}
}

func TestApply_DisableTelemetry(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{}`)

	b := true
	te := config.TargetEntry{Options: config.Options{DisableTelemetry: &b}}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	env := envMap(t, readSettingsMap(t, tgt))
	if env["DISABLE_TELEMETRY"] != "1" {
		t.Errorf("DISABLE_TELEMETRY = %q, want 1", env["DISABLE_TELEMETRY"])
	}
}

// ---------- Discover ----------

func TestDiscover_MissingFile(t *testing.T) {
	tgt := setupClaude(t)
	// Don't write a settings.json — only dir exists.

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr == nil {
		t.Fatal("Discover() returned nil DiscoveryResult")
	}
	if dr.ID != ID {
		t.Errorf("ID = %q, want %q", dr.ID, ID)
	}
}

func TestDiscover_FullSettings(t *testing.T) {
	tgt := setupClaude(t)
	content := `{
		"env": {
			"ANTHROPIC_BASE_URL": "https://custom.api",
			"ANTHROPIC_AUTH_TOKEN": "sk-abc",
			"ANTHROPIC_MODEL": "claude-opus",
			"ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-haiku",
			"DISABLE_TELEMETRY": "1",
			"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1"
		}
	}`
	writeSettings(t, tgt, content)

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr.Provider.Endpoint != "https://custom.api" {
		t.Errorf("Endpoint = %q", dr.Provider.Endpoint)
	}
	if dr.Provider.APIKey != "sk-abc" {
		t.Errorf("APIKey = %q", dr.Provider.APIKey)
	}
	if dr.Provider.Model != "claude-opus" {
		t.Errorf("Model = %q", dr.Provider.Model)
	}
	if dr.Provider.SmallModel != "claude-haiku" {
		t.Errorf("SmallModel = %q", dr.Provider.SmallModel)
	}
	// DISABLE_TELEMETRY and DISABLE_BETAS are Options-level; they are skipped in Discover().
	// They must NOT appear in dr.Env.
	if _, ok := dr.Env["DISABLE_TELEMETRY"]; ok {
		t.Error("DISABLE_TELEMETRY should not appear in DiscoveryResult.Env")
	}
	if _, ok := dr.Env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"]; ok {
		t.Error("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS should not appear in DiscoveryResult.Env")
	}
}

func TestDiscover_CustomEnv(t *testing.T) {
	tgt := setupClaude(t)
	writeSettings(t, tgt, `{"env": {"MY_CUSTOM_VAR": "hello"}}`)

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr.Env["MY_CUSTOM_VAR"] != "hello" {
		t.Errorf("custom env MY_CUSTOM_VAR = %q, want hello", dr.Env["MY_CUSTOM_VAR"])
	}
}

func TestDiscover_AlwaysThinkingSkipped(t *testing.T) {
	tgt := setupClaude(t)
	// alwaysThinkingEnabled is an Options field (context-level); Discover() skips it.
	writeSettings(t, tgt, `{"alwaysThinkingEnabled": true}`)

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	// DiscoveryResult has no Options; verify the result is non-nil and has the right ID.
	if dr == nil || dr.ID != ID {
		t.Errorf("unexpected Discover() result: %v", dr)
	}
}
