package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/IQNeoXen/aictx/internal/config"
	"github.com/IQNeoXen/aictx/internal/target"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var discoverName string

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Detect current config from installed tools and save as a context",
	Args:  cobra.NoArgs,
	RunE:  discoverRun,
}

func init() {
	discoverCmd.Flags().StringVar(&discoverName, "name", "", "Context name (prompts if not provided)")
}

func discoverRun(cmd *cobra.Command, args []string) error {
	fmt.Println("Scanning targets...")

	var results []*config.DiscoveryResult

	for _, t := range target.All() {
		if !t.Detect() {
			fmt.Printf("  [--] %s: not found\n", t.Name())
			continue
		}

		dr, err := t.Discover()
		if err != nil {
			fmt.Printf("  [!!] %s: %v\n", t.Name(), err)
			continue
		}
		if dr == nil {
			fmt.Printf("  [--] %s: no config found\n", t.Name())
			continue
		}

		fmt.Printf("  [OK] %s (%s)\n", t.Name(), t.ID())
		results = append(results, dr)
	}

	if len(results) == 0 {
		return fmt.Errorf("no targets with existing config found")
	}

	// Build the context: lift first non-empty Provider to context level.
	ctx := &config.Context{Name: "<name>"}
	for _, dr := range results {
		if !dr.Provider.IsEmpty() && ctx.Provider.IsEmpty() {
			ctx.Provider = dr.Provider
		}
		ctx.Targets = append(ctx.Targets, config.TargetEntry{
			ID:  dr.ID,
			Env: dr.Env,
		})
	}

	// Warn if multiple targets returned differing non-empty providers.
	var nonEmptyProviders []config.Provider
	for _, dr := range results {
		if !dr.Provider.IsEmpty() {
			nonEmptyProviders = append(nonEmptyProviders, dr.Provider)
		}
	}
	if len(nonEmptyProviders) > 1 {
		first := nonEmptyProviders[0]
		for _, p := range nonEmptyProviders[1:] {
			if p.Endpoint != first.Endpoint || p.APIKey != first.APIKey || p.Model != first.Model {
				fmt.Fprintf(os.Stderr, "aictx: warning: targets returned different providers; using first non-empty provider\n")
				break
			}
		}
	}

	// Print discovered config as YAML (mask secrets in preview).
	preview := *ctx
	if preview.Provider.APIKey != "" {
		preview.Provider.APIKey = maskValue(preview.Provider.APIKey)
	}
	yamlBytes, _ := yaml.Marshal(preview)
	fmt.Printf("\n%s", string(yamlBytes))

	// Get context name
	name := discoverName
	if name == "" {
		fmt.Print("Save as context [name]: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			name = strings.TrimSpace(scanner.Text())
		}
		if name == "" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	ctx.Name = name

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if existing := cfg.FindContext(name); existing != nil {
		return fmt.Errorf("context %q already exists. Remove it first with 'aictx rm %s'", name, name)
	}

	cfg.Contexts = append(cfg.Contexts, *ctx)
	cfg.State.Current = name

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("\nContext \033[1m%s\033[0m saved and set as current.\n", name)
	return nil
}
