package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fschneidewind/aictx/internal/config"
	"github.com/fschneidewind/aictx/internal/target"
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

	var targets []config.TargetEntry

	for _, t := range target.All() {
		if !t.Detect() {
			fmt.Printf("  [--] %s: not found\n", t.Name())
			continue
		}

		te, err := t.Discover()
		if err != nil {
			fmt.Printf("  [!!] %s: %v\n", t.Name(), err)
			continue
		}
		if te == nil {
			fmt.Printf("  [--] %s: no config found\n", t.Name())
			continue
		}

		fmt.Printf("  [OK] %s (%s)\n", t.Name(), t.ID())
		targets = append(targets, *te)
	}

	if len(targets) == 0 {
		return fmt.Errorf("no targets with existing config found")
	}

	result := &config.Context{
		Name:    "<name>",
		Targets: targets,
	}

	// Print discovered config as YAML (mask secrets in preview)
	preview := *result
	for i := range preview.Targets {
		if preview.Targets[i].Provider.APIKey != "" {
			preview.Targets[i].Provider.APIKey = maskValue(preview.Targets[i].Provider.APIKey)
		}
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

	result.Name = name

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if existing := cfg.FindContext(name); existing != nil {
		return fmt.Errorf("context %q already exists. Remove it first with 'aictx rm %s'", name, name)
	}

	cfg.Contexts = append(cfg.Contexts, *result)
	cfg.State.Current = name

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("\nContext \033[1m%s\033[0m saved and set as current.\n", name)
	return nil
}
