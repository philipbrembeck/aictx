package claudecli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fschneidewind/aictx/internal/config"
)

const ID = "claude-code-cli"

type Target struct{}

func New() *Target { return &Target{} }

func (t *Target) ID() string   { return ID }
func (t *Target) Name() string { return "Claude Code CLI" }

func (t *Target) settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func (t *Target) Detect() bool {
	_, err := os.Stat(filepath.Dir(t.settingsPath()))
	return err == nil
}

func (t *Target) Apply(te config.TargetEntry) error {
	env := make(map[string]string)

	if te.Provider.Endpoint != "" {
		env["ANTHROPIC_BASE_URL"] = te.Provider.Endpoint
	}
	if te.Provider.APIKey != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = te.Provider.APIKey
	}
	if te.Provider.Model != "" {
		env["ANTHROPIC_MODEL"] = te.Provider.Model
	}
	if te.Provider.SmallModel != "" {
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = te.Provider.SmallModel
	}
	if te.Options.DisableTelemetry != nil && *te.Options.DisableTelemetry {
		env["DISABLE_TELEMETRY"] = "1"
	}
	if te.Options.DisableBetas != nil && *te.Options.DisableBetas {
		env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"] = "1"
	}

	settings := map[string]interface{}{}
	if len(env) > 0 {
		settings["env"] = env
	}
	if te.Options.AlwaysThinking != nil {
		settings["alwaysThinkingEnabled"] = *te.Options.AlwaysThinking
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling claude settings: %w", err)
	}
	data = append(data, '\n')

	path := t.settingsPath()
	tmp := path + ".aictx-tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing claude settings: %w", err)
	}
	return os.Rename(tmp, path)
}

func (t *Target) Discover() (*config.TargetEntry, error) {
	data, err := os.ReadFile(t.settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading claude settings: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing claude settings: %w", err)
	}

	te := &config.TargetEntry{ID: ID}

	if envRaw, ok := raw["env"].(map[string]interface{}); ok {
		for k, v := range envRaw {
			s, ok := v.(string)
			if !ok {
				continue
			}
			switch k {
			case "ANTHROPIC_BASE_URL":
				te.Provider.Endpoint = s
			case "ANTHROPIC_AUTH_TOKEN":
				te.Provider.APIKey = s
			case "ANTHROPIC_MODEL":
				te.Provider.Model = s
			case "ANTHROPIC_DEFAULT_HAIKU_MODEL":
				te.Provider.SmallModel = s
			case "DISABLE_TELEMETRY":
				if s == "1" {
					b := true
					te.Options.DisableTelemetry = &b
				}
			case "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS":
				if s == "1" {
					b := true
					te.Options.DisableBetas = &b
				}
			}
		}
	}

	if thinking, ok := raw["alwaysThinkingEnabled"].(bool); ok {
		te.Options.AlwaysThinking = &thinking
	}

	return te, nil
}
