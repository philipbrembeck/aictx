package cmd

import (
	"fmt"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var showReveal bool

var showCmd = &cobra.Command{
	Use:               "show [name]",
	Short:             "Show context details (defaults to current)",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: contextCompletion,
	RunE:              showRun,
}

func init() {
	showCmd.Flags().BoolVar(&showReveal, "reveal", false, "Show full secret values")
}

func showRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := cfg.State.Current
	if len(args) == 1 {
		name = args[0]
	}
	if name == "" {
		return fmt.Errorf("no context specified and no current context set")
	}

	ctx := cfg.FindContext(name)
	if ctx == nil {
		return fmt.Errorf("context %q not found", name)
	}

	// Print as YAML, masking secrets unless --reveal
	display := *ctx
	if !showReveal {
		for i := range display.Targets {
			if display.Targets[i].Provider.APIKey != "" {
				display.Targets[i].Provider.APIKey = maskValue(display.Targets[i].Provider.APIKey)
			}
		}
	}

	yamlBytes, _ := yaml.Marshal(display)
	fmt.Print(string(yamlBytes))
	return nil
}

func maskValue(v string) string {
	if len(v) <= 8 {
		return "***"
	}
	return v[:8] + "***"
}
