package picli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IQNeoXen/aictx/internal/config"
)

// setupPi creates a temp HOME with the .pi/agent directory and returns a fresh Target.
func setupPi(t *testing.T) *Target {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0755); err != nil {
		t.Fatalf("MkdirAll .pi/agent: %v", err)
	}
	return New()
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

func readExtension(t *testing.T, tgt *Target) string {
	t.Helper()
	data, err := os.ReadFile(tgt.extensionPath())
	if err != nil {
		t.Fatalf("readExtension: %v", err)
	}
	return string(data)
}

// ---------- Detect ----------

func TestDetect_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tgt := New()
	if tgt.Detect() {
		t.Error("Detect() = true, want false when .pi/agent dir missing")
	}
}

func TestDetect_Installed(t *testing.T) {
	tgt := setupPi(t)
	if !tgt.Detect() {
		t.Error("Detect() = false, want true when .pi/agent dir exists")
	}
}

// ---------- Apply ----------

func TestApply_BasicProvider(t *testing.T) {
	tgt := setupPi(t)

	te := config.TargetEntry{
		Provider: config.Provider{
			Endpoint: "https://aikeys.maibornwolff.de/",
			APIKey:   "sk-test-key",
			Model:    "claude-sonnet-4-6",
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	// Check extension file
	ext := readExtension(t, tgt)
	if !strings.Contains(ext, `"https://aikeys.maibornwolff.de/"`) {
		t.Error("extension missing baseUrl")
	}
	if !strings.Contains(ext, `"sk-test-key"`) {
		t.Error("extension missing apiKey")
	}
	// Provider name is derived from the endpoint hostname ("aikeys" from aikeys.maibornwolff.de)
	// so pi does not apply its stored Anthropic OAuth credentials to proxy requests.
	if !strings.Contains(ext, `"aikeys"`) {
		t.Error("extension should register under derived provider name, not \"anthropic\"")
	}

	// Check settings
	m := readSettingsMap(t, tgt)
	if m["defaultModel"] != "claude-sonnet-4-6" {
		t.Errorf("defaultModel = %v", m["defaultModel"])
	}
	if m["defaultProvider"] != "aikeys" {
		t.Errorf("defaultProvider = %v", m["defaultProvider"])
	}
}

func TestApply_EmptyProvider_RemovesExtension(t *testing.T) {
	tgt := setupPi(t)

	// First apply with provider
	te := config.TargetEntry{
		Provider: config.Provider{
			Endpoint: "https://example.com",
			APIKey:   "sk-xxx",
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() with provider: %v", err)
	}
	if _, err := os.Stat(tgt.extensionPath()); err != nil {
		t.Fatal("extension file should exist after Apply with provider")
	}

	// Apply with empty provider (OAuth mode)
	if err := tgt.Apply(config.TargetEntry{}); err != nil {
		t.Fatalf("Apply() empty: %v", err)
	}
	if _, err := os.Stat(tgt.extensionPath()); !os.IsNotExist(err) {
		t.Error("extension file should be removed for empty provider (OAuth)")
	}
}

func TestApply_Headers(t *testing.T) {
	tgt := setupPi(t)

	te := config.TargetEntry{
		Provider: config.Provider{
			Endpoint: "https://proxy.example.com",
			APIKey:   "sk-test",
			Headers:  map[string]string{"X-Team-ID": "eng"},
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	ext := readExtension(t, tgt)
	if !strings.Contains(ext, "X-Team-ID") {
		t.Error("extension missing header key")
	}
	if !strings.Contains(ext, "eng") {
		t.Error("extension missing header value")
	}
}

func TestApply_AlwaysThinking(t *testing.T) {
	tgt := setupPi(t)

	b := true
	te := config.TargetEntry{
		Options: config.Options{AlwaysThinking: &b},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if m["defaultThinkingLevel"] != "medium" {
		t.Errorf("defaultThinkingLevel = %v, want medium", m["defaultThinkingLevel"])
	}
}

func TestApply_MergesExistingSettings(t *testing.T) {
	tgt := setupPi(t)

	// Write existing settings
	existing := `{"lastChangelogVersion": "0.65.2"}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	te := config.TargetEntry{
		Provider: config.Provider{Model: "claude-sonnet-4-6"},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	m := readSettingsMap(t, tgt)
	if m["lastChangelogVersion"] != "0.65.2" {
		t.Error("existing setting was lost")
	}
	if m["defaultModel"] != "claude-sonnet-4-6" {
		t.Errorf("defaultModel = %v", m["defaultModel"])
	}
}

// ---------- Discover ----------

func TestDiscover_NoExtension(t *testing.T) {
	tgt := setupPi(t)

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr.Provider.Endpoint != "" {
		t.Errorf("Endpoint = %q, want empty", dr.Provider.Endpoint)
	}
}

func TestDiscover_WithExtensionAndSettings(t *testing.T) {
	tgt := setupPi(t)

	// Write settings (defaultThinkingLevel is Options-level; skipped in Discover)
	settings := `{"defaultModel": "claude-sonnet-4-6", "defaultThinkingLevel": "medium"}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(settings), 0644); err != nil {
		t.Fatal(err)
	}

	// Write extension
	ext := `import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
export default function (pi: ExtensionAPI) {
  pi.registerProvider("anthropic", {
    baseUrl: "https://aikeys.maibornwolff.de/",
    apiKey: "sk-test-key",
  });
}`
	if err := os.MkdirAll(tgt.extensionsDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tgt.extensionPath(), []byte(ext), 0644); err != nil {
		t.Fatal(err)
	}

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr.Provider.Endpoint != "https://aikeys.maibornwolff.de/" {
		t.Errorf("Endpoint = %q", dr.Provider.Endpoint)
	}
	if dr.Provider.APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q", dr.Provider.APIKey)
	}
	if dr.Provider.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q", dr.Provider.Model)
	}
	// AlwaysThinking is Options-level (context-level); DiscoveryResult has no Options field.
	// Verify we get a valid result without panicking.
	if dr.ID != ID {
		t.Errorf("ID = %q, want %q", dr.ID, ID)
	}
}
