package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fschneidewind/aictx/internal/config"
	"github.com/fschneidewind/aictx/internal/target"
	"github.com/spf13/cobra"
)

var (
	addTargets     []string
	addDesc        string
	addEndpoint    string
	addAPIKey      string
	addModel       string
	addSmallModel  string
	addThinking    bool
	addNoTelemetry bool
	addNoBetas     bool
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new context",
	Args:  cobra.ExactArgs(1),
	RunE:  addRun,
}

func init() {
	addCmd.Flags().StringArrayVar(&addTargets, "target", nil, "Target to include (repeatable, e.g. claude-code-cli)")
	addCmd.Flags().StringVar(&addDesc, "description", "", "Context description")
	addCmd.Flags().StringVar(&addEndpoint, "endpoint", "", "Provider endpoint URL")
	addCmd.Flags().StringVar(&addAPIKey, "api-key", "", "Provider API key")
	addCmd.Flags().StringVar(&addModel, "model", "", "Model (e.g. claude-opus-4.6)")
	addCmd.Flags().StringVar(&addSmallModel, "small-model", "", "Small/cheap model (e.g. claude-haiku-4.5)")
	addCmd.Flags().BoolVar(&addThinking, "thinking", false, "Enable always thinking")
	addCmd.Flags().BoolVar(&addNoTelemetry, "no-telemetry", false, "Disable telemetry")
	addCmd.Flags().BoolVar(&addNoBetas, "no-betas", false, "Disable experimental betas")
}

func addRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.FindContext(name) != nil {
		return fmt.Errorf("context %q already exists", name)
	}

	ctx := config.Context{Name: name}

	flagsProvided := len(addTargets) > 0 || addDesc != "" || addEndpoint != "" ||
		addAPIKey != "" || addModel != "" || addSmallModel != "" ||
		addThinking || addNoTelemetry || addNoBetas

	if flagsProvided {
		ctx.Description = addDesc

		// Build provider and options from flags (same for all targets in flag mode)
		prov := config.Provider{
			Endpoint:   addEndpoint,
			APIKey:     addAPIKey,
			Model:      addModel,
			SmallModel: addSmallModel,
		}
		var opts config.Options
		if addThinking {
			b := true
			opts.AlwaysThinking = &b
		}
		if addNoTelemetry {
			b := true
			opts.DisableTelemetry = &b
		}
		if addNoBetas {
			b := true
			opts.DisableBetas = &b
		}

		for _, tid := range addTargets {
			if target.ByID(tid) == nil {
				return fmt.Errorf("unknown target %q. Available: %v", tid, target.IDs())
			}
			ctx.Targets = append(ctx.Targets, config.TargetEntry{
				ID:       tid,
				Provider: prov,
				Options:  opts,
			})
		}
	} else {
		// Interactive mode
		scanner := bufio.NewScanner(os.Stdin)

		ctx.Description = prompt(scanner, "Description")

		// Target selection
		fmt.Println("\nAvailable targets:")
		allTargets := target.All()
		for i, t := range allTargets {
			detected := ""
			if t.Detect() {
				detected = " (detected)"
			}
			fmt.Printf("  [%d] %s (%s)%s\n", i+1, t.Name(), t.ID(), detected)
		}
		fmt.Print("Select targets (comma-separated numbers, e.g. 1,2): ")

		var selectedTargets []target.Target
		if scanner.Scan() {
			for _, p := range strings.Split(scanner.Text(), ",") {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				idx := 0
				fmt.Sscanf(p, "%d", &idx)
				if idx >= 1 && idx <= len(allTargets) {
					selectedTargets = append(selectedTargets, allTargets[idx-1])
				}
			}
		}
		if len(selectedTargets) == 0 {
			return fmt.Errorf("no targets selected")
		}

		// Configure each target
		for _, t := range selectedTargets {
			fmt.Printf("\n--- %s (%s) ---\n", t.Name(), t.ID())

			te := config.TargetEntry{ID: t.ID()}

			fmt.Println("Provider (leave empty for native auth / OAuth):")
			te.Provider.Endpoint = prompt(scanner, "  Endpoint URL")
			te.Provider.APIKey = prompt(scanner, "  API Key")
			te.Provider.Model = prompt(scanner, "  Model (e.g. claude-opus-4.6)")
			te.Provider.SmallModel = prompt(scanner, "  Small model (e.g. claude-haiku-4.5)")

			fmt.Println("Options:")
			if yesNo(scanner, "  Always thinking?", true) {
				b := true
				te.Options.AlwaysThinking = &b
			}
			if yesNo(scanner, "  Disable telemetry?", true) {
				b := true
				te.Options.DisableTelemetry = &b
			}
			if yesNo(scanner, "  Disable experimental betas?", false) {
				b := true
				te.Options.DisableBetas = &b
			}

			ctx.Targets = append(ctx.Targets, te)
		}
	}

	cfg.Contexts = append(cfg.Contexts, ctx)
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Context \033[1m%s\033[0m added.\n", name)
	return nil
}

func prompt(scanner *bufio.Scanner, label string) string {
	fmt.Printf("%s: ", label)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func yesNo(scanner *bufio.Scanner, question string, defaultYes bool) bool {
	hint := "Y/n"
	if !defaultYes {
		hint = "y/N"
	}
	fmt.Printf("%s (%s): ", question, hint)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "" {
			return defaultYes
		}
		return answer == "y" || answer == "yes"
	}
	return defaultYes
}
