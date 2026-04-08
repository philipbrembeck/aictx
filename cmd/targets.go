package cmd

import (
	"fmt"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/picker"
	"github.com/IQNeoXen/aictx/internal/target"
	"github.com/spf13/cobra"
)

var targetsCmd = &cobra.Command{
	Use:   "targets [contextname]",
	Short: "Add or remove targets from a context",
	Long: `Show a checkbox picker to add or remove targets from a context.
If no context name is provided, the current context is used.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  targetsRun,
}

func targetsRun(cmd *cobra.Command, args []string) error {
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

	allTargets := target.All()
	labels := make([]string, len(allTargets))
	initialSelected := make([]bool, len(allTargets))
	for i, t := range allTargets {
		lbl := fmt.Sprintf("%s (%s)", t.Name(), t.ID())
		if t.Detect() {
			lbl += " [detected]"
		}
		labels[i] = lbl
		initialSelected[i] = ctx.HasTarget(t.ID())
	}

	var result []bool
	if picker.IsTerminal() {
		result, err = picker.PickMulti(labels, initialSelected)
		if err != nil {
			return err
		}
		if result == nil {
			return nil // cancelled
		}
	} else {
		// Non-terminal fallback: display current state and exit
		fmt.Printf("Targets for context %q:\n", name)
		for i, lbl := range labels {
			checked := " "
			if initialSelected[i] {
				checked = "x"
			}
			fmt.Printf("  [%s] %s\n", checked, lbl)
		}
		fmt.Println("(non-interactive: use 'aictx add' or edit config.yaml to modify targets)")
		return nil
	}

	// Rebuild target list preserving existing TargetEntry data (env vars, etc.)
	var newTargets []config.TargetEntry
	for i, t := range allTargets {
		if !result[i] {
			continue
		}
		if existing := ctx.GetTarget(t.ID()); existing != nil {
			newTargets = append(newTargets, *existing)
		} else {
			newTargets = append(newTargets, config.TargetEntry{ID: t.ID()})
		}
	}
	ctx.Targets = newTargets

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Saved targets for context %q.\n", name)
	return nil
}
