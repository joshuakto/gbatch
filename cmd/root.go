package cmd

import (
	"fmt"
	"os"

	"github.com/joshuakto/gbatch/internal/config"
	"github.com/joshuakto/gbatch/internal/gcloud"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var (
	jsonOutput bool
	cfg        *config.Config
	executor   gcloud.Executor
)

var rootCmd = &cobra.Command{
	Use:   "gbatch",
	Short: "Lightweight job scheduler for Google Cloud",
	Long: `gBatch — A lightweight job scheduler CLI for Google Cloud.

Submit, monitor, and manage compute workflows with a familiar CLI.
Replaces UGER with a modern, cost-aware experience on GCP.

Quick start:
  gbatch submit job.sh           Submit a job
  gbatch status                  Check job status
  gbatch cost --today            See today's spend

First time? Run:
  gbatch doctor                  Check your GCP setup`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "output-json", "o", false, "Output as JSON")
	rootCmd.Version = Version

	cobra.OnInitialize(func() {
		cfg = config.Load()
	})
}

// initExecutor lazily creates the gcloud executor, printing a helpful error if gcloud is missing.
func initExecutor() error {
	if executor != nil {
		return nil
	}
	exec, err := gcloud.NewExecutor()
	if err != nil {
		return fmt.Errorf("gcloud not found.\n  Install: https://cloud.google.com/sdk/docs/install\n  Then run: gcloud auth application-default login")
	}
	executor = exec
	return nil
}
