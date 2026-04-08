package cmd

import (
	"fmt"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/keyring"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:               "rename <old-name> <new-name>",
	Aliases:           []string{"mv"},
	Short:             "Rename a context",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: renameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if oldName == newName {
			return fmt.Errorf("old and new names are the same")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx := cfg.FindContext(oldName)
		if ctx == nil {
			return fmt.Errorf("context %q not found", oldName)
		}
		if cfg.FindContext(newName) != nil {
			return fmt.Errorf("context %q already exists", newName)
		}

		hasKeyring := ctx.HasKeyringKey

		cfg.RenameContext(oldName, newName)

		// Save — for contexts with in-memory API keys, Save will write new
		// keyring entries under the new context name automatically.
		if err := config.Save(cfg); err != nil {
			return err
		}

		// Delete the stale keyring entry stored under the old name.
		if hasKeyring {
			if kerr := keyring.Delete(oldName); kerr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "aictx: warning: could not delete old keychain entry for %s: %v\n", oldName, kerr)
			}
		}

		fmt.Printf("Context \033[1m%s\033[0m renamed to \033[1m%s\033[0m.\n", oldName, newName)
		return nil
	},
}

func renameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cfg.ContextNames(), cobra.ShellCompDirectiveNoFileComp
}
