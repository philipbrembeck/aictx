package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/spf13/cobra"
)

var (
	currentJSON   bool
	currentEnv    bool
	currentReveal bool
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current context name",
	Args:  cobra.NoArgs,
	RunE:  currentRun,
}

func init() {
	currentCmd.Flags().BoolVar(&currentJSON, "json", false, "Print active context as JSON")
	currentCmd.Flags().BoolVar(&currentEnv, "env", false, "Print active context as export KEY=value lines")
	currentCmd.Flags().BoolVar(&currentReveal, "reveal", false, "Show full secret values (use with --json or --env)")
}

func currentRun(cmd *cobra.Command, args []string) error {
	if currentJSON && currentEnv {
		return fmt.Errorf("flags --json and --env are mutually exclusive")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.State.Current == "" {
		fmt.Println("(none)")
		return nil
	}

	if !currentJSON && !currentEnv {
		fmt.Println(cfg.State.Current)
		return nil
	}

	ctx := cfg.FindContext(cfg.State.Current)
	if ctx == nil {
		return fmt.Errorf("current context %q not found in config", cfg.State.Current)
	}

	if currentJSON {
		return currentPrintJSON(ctx)
	}

	// --env
	return currentPrintEnv(ctx)
}

// jsonTarget is the per-target shape for --json output.
type jsonTarget struct {
	ID         string `json:"id"`
	Endpoint   string `json:"endpoint,omitempty"`
	APIKey     string `json:"apiKey,omitempty"`
	Model      string `json:"model,omitempty"`
	SmallModel string `json:"smallModel,omitempty"`
}

// jsonContext is the top-level shape for --json output.
type jsonContext struct {
	Name    string       `json:"name"`
	Targets []jsonTarget `json:"targets"`
}

func currentPrintJSON(ctx *config.Context) error {
	out := jsonContext{
		Name:    ctx.Name,
		Targets: make([]jsonTarget, 0, len(ctx.Targets)),
	}
	for _, te := range ctx.Targets {
		apiKey := te.Provider.APIKey
		if apiKey != "" && !currentReveal {
			apiKey = maskValue(apiKey)
		}
		out.Targets = append(out.Targets, jsonTarget{
			ID:         te.ID,
			Endpoint:   te.Provider.Endpoint,
			APIKey:     apiKey,
			Model:      te.Provider.Model,
			SmallModel: te.Provider.SmallModel,
		})
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func currentPrintEnv(ctx *config.Context) error {
	for _, te := range ctx.Targets {
		if te.Provider.APIKey != "" {
			apiKey := te.Provider.APIKey
			if !currentReveal {
				apiKey = maskValue(apiKey)
			}
			fmt.Printf("export ANTHROPIC_AUTH_TOKEN=%q\n", apiKey)
		}
		if te.Provider.Endpoint != "" {
			fmt.Printf("export ANTHROPIC_BASE_URL=%q\n", te.Provider.Endpoint)
		}
		if te.Provider.Model != "" {
			fmt.Printf("export ANTHROPIC_MODEL=%q\n", te.Provider.Model)
		}
		if te.Provider.SmallModel != "" {
			fmt.Printf("export ANTHROPIC_DEFAULT_HAIKU_MODEL=%q\n", te.Provider.SmallModel)
		}
		if te.Options.DisableTelemetry != nil && *te.Options.DisableTelemetry {
			fmt.Printf("export DISABLE_TELEMETRY=%q\n", "1")
		}
		if te.Options.DisableBetas != nil && *te.Options.DisableBetas {
			fmt.Printf("export CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=%q\n", "1")
		}
	}
	return nil
}
