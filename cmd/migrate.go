package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuakto/gbatch/internal/migrate"
	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var migrateDir string

var migrateCmd = &cobra.Command{
	Use:   "migrate [script]",
	Short: "Convert UGER qsub scripts to gBatch format",
	Long: `Parse UGER submission scripts and convert #$ directives to gbatch commands.

Examples:
  gbatch migrate align.sh              Convert a single script
  gbatch migrate --dir ./scripts/      Convert all scripts in a directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateDir, "dir", "", "Convert all scripts in a directory")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if migrateDir != "" {
		return migrateDirectory(migrateDir)
	}

	if len(args) == 0 {
		output.ErrorHint(
			"No script specified",
			"Usage: gbatch migrate script.sh or gbatch migrate --dir ./scripts/",
		)
		return nil
	}

	return migrateFile(args[0])
}

func migrateFile(path string) error {
	result, err := migrate.ParseFile(path)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Migration failed: %v", err),
			"Is this a UGER/SGE submission script with #$ directives?",
		)
		return nil
	}

	if jsonOutput {
		return output.JSON(result)
	}

	// Show warnings
	for _, w := range result.Warnings {
		output.Warn(w)
	}

	// Show converted command
	gbatchCmd := result.ToGbatchCommand()
	fmt.Println()
	output.Success("Converted command:")
	fmt.Printf("  %s\n", output.Bold_(gbatchCmd))
	fmt.Println()
	fmt.Printf("%s Found %d UGER directives", output.Dim_("│"), len(result.Directives))
	if len(result.Warnings) > 0 {
		fmt.Printf(", %d unsupported (see warnings above)", len(result.Warnings))
	}
	fmt.Println()

	return nil
}

func migrateDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Cannot read directory: %s", dir),
			"Check the path and permissions",
		)
		return nil
	}

	converted := 0
	failed := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		result, err := migrate.ParseFile(path)
		if err != nil {
			failed++
			continue
		}
		converted++
		fmt.Printf("%s %s → %s\n", output.SuccessPrefix(), output.Dim_(entry.Name()), result.ToGbatchCommand())
	}

	fmt.Println()
	if converted > 0 {
		output.Success(fmt.Sprintf("Converted %d scripts", converted))
	}
	if failed > 0 {
		output.Warn(fmt.Sprintf("%d files skipped (no UGER directives found)", failed))
	}

	return nil
}
