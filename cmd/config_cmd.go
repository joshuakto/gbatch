package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Manage .gbatchrc configuration",
	Long: `View or set gBatch configuration values.

Examples:
  gbatch config                    Show current config
  gbatch config project my-proj    Set project
  gbatch config region us-east1    Set region
  gbatch config default_cpus 8     Set default CPUs`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return showConfig()
	}
	if len(args) == 1 {
		return getConfig(args[0])
	}
	return setConfig(args[0], args[1])
}

func showConfig() error {
	if jsonOutput {
		return output.JSON(cfg)
	}

	fmt.Println(output.Bold_("Current configuration:"))
	fmt.Printf("  project:      %s\n", valueOrDefault(cfg.Project, "(not set)"))
	fmt.Printf("  region:       %s\n", cfg.Region)
	fmt.Printf("  default_cpus: %d\n", cfg.DefaultCPUs)
	fmt.Printf("  default_mem:  %s\n", cfg.DefaultMem)
	if len(cfg.Mounts) > 0 {
		fmt.Printf("  mounts:       %v\n", cfg.Mounts)
	}

	// Show source files
	fmt.Println()
	if _, err := os.Stat(".gbatchrc"); err == nil {
		output.Success("Project config: .gbatchrc")
	}
	home, _ := os.UserHomeDir()
	userCfg := filepath.Join(home, ".gbatch", "config.yaml")
	if _, err := os.Stat(userCfg); err == nil {
		output.Success(fmt.Sprintf("User config: %s", userCfg))
	}

	return nil
}

func getConfig(key string) error {
	switch key {
	case "project":
		fmt.Println(cfg.Project)
	case "region":
		fmt.Println(cfg.Region)
	case "default_cpus":
		fmt.Println(cfg.DefaultCPUs)
	case "default_mem":
		fmt.Println(cfg.DefaultMem)
	default:
		output.ErrorHint(
			fmt.Sprintf("Unknown config key: %s", key),
			"Valid keys: project, region, default_cpus, default_mem",
		)
	}
	return nil
}

func setConfig(key, value string) error {
	// Read existing .gbatchrc or create new
	data := make(map[string]any)
	if existing, err := os.ReadFile(".gbatchrc"); err == nil {
		yaml.Unmarshal(existing, &data)
	}

	data[key] = value

	out, err := yaml.Marshal(data)
	if err != nil {
		output.Error(fmt.Sprintf("Failed to serialize config: %v", err))
		return nil
	}

	if err := os.WriteFile(".gbatchrc", out, 0644); err != nil {
		output.Error(fmt.Sprintf("Failed to write .gbatchrc: %v", err))
		return nil
	}

	output.Success(fmt.Sprintf("Set %s = %s in .gbatchrc", key, value))
	return nil
}

func valueOrDefault(v, def string) string {
	if v == "" {
		return output.Dim_(def)
	}
	return v
}
