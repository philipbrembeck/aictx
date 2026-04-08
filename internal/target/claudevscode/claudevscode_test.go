package claudevscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/tidwall/gjson"
)

// vscodeSettingsDir returns the platform-specific path for settings.json relative
// to a given home directory — mirrors the logic in settingsPath().
func vscodeSettingsDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		return filepath.Join(home, ".config", "Code", "User")
	default:
		// Windows: settingsPath() uses APPDATA; we skip platform-specific test there.
		return filepath.Join(home, "AppData", "Roaming", "Code", "User")
	}
}

// setupVSCode creates a temp HOME with a minimal VSCode settings.json and returns a fresh Target.
func setupVSCode(t *testing.T) *Target {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	}

	dir := vscodeSettingsDir(home)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll vscode dir: %v", err)
	}
	settingsFile := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("writeFile settings.json: %v", err)
	}
	return New()
}

func readVSCodeSettings(t *testing.T, tgt *Target) []byte {
	t.Helper()
	data, err := os.ReadFile(tgt.settingsPath())
	if err != nil {
		t.Fatalf("readVSCodeSettings: %v", err)
	}
	return data
}

// ---------- buildEnvVarsArray ----------

func TestBuildEnvVarsArray_Empty(t *testing.T) {
	result := buildEnvVarsArray(map[string]string{})
	if result != nil {
		t.Errorf("buildEnvVarsArray(empty) = %v, want nil", result)
	}
}

func TestBuildEnvVarsArray_Sorted(t *testing.T) {
	input := map[string]string{
		"Z_VAR": "z",
		"A_VAR": "a",
		"M_VAR": "m",
	}
	result := buildEnvVarsArray(input)
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0].Name != "A_VAR" || result[1].Name != "M_VAR" || result[2].Name != "Z_VAR" {
		t.Errorf("not sorted: %v", result)
	}
}

// ---------- Detect ----------

func TestDetect_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tgt := New()
	if tgt.Detect() {
		t.Error("Detect() = true, want false when settings.json absent")
	}
}

func TestDetect_Installed(t *testing.T) {
	tgt := setupVSCode(t)
	if !tgt.Detect() {
		t.Error("Detect() = false, want true when settings.json exists")
	}
}

// ---------- Apply ----------

func TestApply_WritesEnvVars(t *testing.T) {
	tgt := setupVSCode(t)

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

	data := readVSCodeSettings(t, tgt)
	arr := gjson.GetBytes(data, `claudeCode\.environmentVariables`)
	if !arr.IsArray() {
		t.Fatal("claudeCode.environmentVariables is not an array")
	}

	envMap := map[string]string{}
	arr.ForEach(func(_, v gjson.Result) bool {
		envMap[v.Get("name").String()] = v.Get("value").String()
		return true
	})

	if envMap["ANTHROPIC_BASE_URL"] != "https://api.example.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %q", envMap["ANTHROPIC_BASE_URL"])
	}
	if envMap["ANTHROPIC_AUTH_TOKEN"] != "sk-test" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %q", envMap["ANTHROPIC_AUTH_TOKEN"])
	}
}

func TestApply_PreservesOtherSettings(t *testing.T) {
	tgt := setupVSCode(t)
	if err := os.WriteFile(tgt.settingsPath(), []byte(`{"editor.fontSize": 14}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "m"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	fontSize := gjson.GetBytes(data, "editor\\.fontSize")
	if fontSize.Int() != 14 {
		t.Errorf("editor.fontSize = %v, want 14", fontSize)
	}
}

func TestApply_MergesExistingEnvVars(t *testing.T) {
	tgt := setupVSCode(t)
	existing := `{
		"claudeCode.environmentVariables": [
			{"name": "EXISTING_VAR", "value": "kept"}
		]
	}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "new"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	arr := gjson.GetBytes(data, `claudeCode\.environmentVariables`)
	envMap := map[string]string{}
	arr.ForEach(func(_, v gjson.Result) bool {
		envMap[v.Get("name").String()] = v.Get("value").String()
		return true
	})

	if envMap["EXISTING_VAR"] != "kept" {
		t.Error("EXISTING_VAR was removed, should have been preserved")
	}
	if envMap["ANTHROPIC_MODEL"] != "new" {
		t.Errorf("ANTHROPIC_MODEL = %q, want new", envMap["ANTHROPIC_MODEL"])
	}
}

func TestApply_CleansUpPrevEnvKeys(t *testing.T) {
	tgt := setupVSCode(t)
	existing := `{
		"claudeCode.environmentVariables": [
			{"name": "ANTHROPIC_AUTH_TOKEN", "value": "old-key"},
			{"name": "ANTHROPIC_MODEL", "value": "old-model"}
		]
	}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	tgt.PrevEnvKeys = []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_MODEL"}

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "new-model"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	arr := gjson.GetBytes(data, `claudeCode\.environmentVariables`)
	envMap := map[string]string{}
	arr.ForEach(func(_, v gjson.Result) bool {
		envMap[v.Get("name").String()] = v.Get("value").String()
		return true
	})

	if _, ok := envMap["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Error("ANTHROPIC_AUTH_TOKEN should have been removed (in PrevEnvKeys)")
	}
	if envMap["ANTHROPIC_MODEL"] != "new-model" {
		t.Errorf("ANTHROPIC_MODEL = %q, want new-model", envMap["ANTHROPIC_MODEL"])
	}
}

func TestApply_SetsSelectedModel(t *testing.T) {
	tgt := setupVSCode(t)

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "claude-opus"}}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	model := gjson.GetBytes(data, `claudeCode\.selectedModel`)
	if model.String() != "claude-opus" {
		t.Errorf("selectedModel = %q, want claude-opus", model.String())
	}
}

func TestApply_DeletesSelectedModelWhenEmpty(t *testing.T) {
	tgt := setupVSCode(t)
	if err := os.WriteFile(tgt.settingsPath(), []byte(`{"claudeCode.selectedModel": "old"}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Apply with no model.
	if err := tgt.Apply(config.TargetEntry{}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	model := gjson.GetBytes(data, `claudeCode\.selectedModel`)
	if model.Exists() {
		t.Errorf("selectedModel should be absent, got %q", model.String())
	}
}

func TestApply_DeletesEnvVarsWhenEmpty(t *testing.T) {
	tgt := setupVSCode(t)
	if err := os.WriteFile(tgt.settingsPath(), []byte(`{
		"claudeCode.environmentVariables": [{"name": "ANTHROPIC_MODEL", "value": "old"}]
	}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	tgt.PrevEnvKeys = []string{"ANTHROPIC_MODEL"}

	// Apply with no provider — nothing to write.
	if err := tgt.Apply(config.TargetEntry{}); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	arr := gjson.GetBytes(data, `claudeCode\.environmentVariables`)
	if arr.Exists() {
		t.Errorf("environmentVariables should be absent when empty, got %v", arr)
	}
}

func TestApply_TrailingCommaHandled(t *testing.T) {
	tgt := setupVSCode(t)
	// Trailing comma as VSCode allows.
	if err := os.WriteFile(tgt.settingsPath(), []byte(`{"editor.fontSize": 12,}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := tgt.Apply(config.TargetEntry{Provider: config.Provider{Model: "m"}}); err != nil {
		t.Fatalf("Apply() with trailing comma: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	fontSize := gjson.GetBytes(data, "editor\\.fontSize")
	if fontSize.Int() != 12 {
		t.Errorf("editor.fontSize = %v after trailing-comma handling", fontSize)
	}
}

func TestApply_Headers(t *testing.T) {
	tgt := setupVSCode(t)

	te := config.TargetEntry{
		Provider: config.Provider{
			Headers: map[string]string{"X-Auth": "tok"},
		},
	}
	if err := tgt.Apply(te); err != nil {
		t.Fatalf("Apply() error: %v", err)
	}

	data := readVSCodeSettings(t, tgt)
	arr := gjson.GetBytes(data, `claudeCode\.environmentVariables`)
	envMap := map[string]string{}
	arr.ForEach(func(_, v gjson.Result) bool {
		envMap[v.Get("name").String()] = v.Get("value").String()
		return true
	})

	raw := envMap["ANTHROPIC_CUSTOM_HEADERS"]
	if raw == "" {
		t.Fatal("ANTHROPIC_CUSTOM_HEADERS not set")
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		t.Fatalf("unmarshal headers: %v", err)
	}
	if headers["X-Auth"] != "tok" {
		t.Errorf("X-Auth = %q", headers["X-Auth"])
	}
}

// ---------- Discover ----------

func TestDiscover_ReadsEnvVars(t *testing.T) {
	tgt := setupVSCode(t)
	content := `{
		"claudeCode.environmentVariables": [
			{"name": "ANTHROPIC_BASE_URL", "value": "https://custom.api"},
			{"name": "ANTHROPIC_AUTH_TOKEN", "value": "sk-disc"},
			{"name": "ANTHROPIC_MODEL", "value": "claude-opus"},
			{"name": "DISABLE_TELEMETRY", "value": "1"}
		],
		"claudeCode.selectedModel": "claude-opus"
	}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr == nil {
		t.Fatal("Discover() returned nil")
	}
	if dr.Provider.Endpoint != "https://custom.api" {
		t.Errorf("Endpoint = %q", dr.Provider.Endpoint)
	}
	if dr.Provider.APIKey != "sk-disc" {
		t.Errorf("APIKey = %q", dr.Provider.APIKey)
	}
	if dr.Provider.Model != "claude-opus" {
		t.Errorf("Model = %q", dr.Provider.Model)
	}
	// DISABLE_TELEMETRY is Options-level; it must not appear in dr.Env.
	if _, ok := dr.Env["DISABLE_TELEMETRY"]; ok {
		t.Error("DISABLE_TELEMETRY should not appear in DiscoveryResult.Env")
	}
}

func TestDiscover_CustomEnv(t *testing.T) {
	tgt := setupVSCode(t)
	content := `{
		"claudeCode.environmentVariables": [
			{"name": "MY_CUSTOM", "value": "val"}
		]
	}`
	if err := os.WriteFile(tgt.settingsPath(), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr.Env["MY_CUSTOM"] != "val" {
		t.Errorf("Env[MY_CUSTOM] = %q, want val", dr.Env["MY_CUSTOM"])
	}
}

func TestDiscover_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tgt := New()

	dr, err := tgt.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if dr != nil {
		t.Errorf("Discover() = %v, want nil when not installed", dr)
	}
}
