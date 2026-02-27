package cmd

import (
	"fmt"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/keyring"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:               "rm <name>",
	Aliases:           []string{"remove", "delete"},
	Short:             "Remove a context",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: contextCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx := cfg.FindContext(name)
		if ctx == nil {
			return fmt.Errorf("context %q not found", name)
		}

		// Delete keyring entries before removing the context from config.
		for _, te := range ctx.Targets {
			if te.HasKeyringKey {
				if kerr := keyring.Delete(name, te.ID); kerr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "aictx: warning: could not delete keychain entry for %s/%s: %v\n", name, te.ID, kerr)
				}
			}
		}

		cfg.RemoveContext(name)

		if cfg.State.Current == name {
			cfg.State.Current = ""
		}
		if cfg.State.Previous == name {
			cfg.State.Previous = ""
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("Context \033[1m%s\033[0m removed.\n", name)
		return nil
	},
}
