package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var logsFollow bool

var logsCmd = &cobra.Command{
	Use:   "logs [job-id]",
	Short: "Stream job logs",
	Long: `View or stream logs for a batch job.

Examples:
  gbatch logs j-4821           View logs
  gbatch logs j-4821 --follow  Stream logs in real-time`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Stream logs in real-time")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	jobID := args[0]
	output.Info(fmt.Sprintf("Fetching logs for %s...", jobID))

	// Build the log filter for this batch job
	filter := fmt.Sprintf("resource.type=cloud_batch_task AND labels.job_uid=%s", jobID)

	gcloudArgs := []string{
		"logging", "read", filter,
		"--order=asc",
		"--limit=200",
	}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	raw, err := executor.RunRaw(context.Background(), gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Could not fetch logs: %v", err),
			"Logs may take 30-60s to appear for new jobs",
		)
		return nil
	}

	if len(raw) == 0 {
		output.Warn("No logs yet. Logs typically appear 30-60s after job starts.")
		return nil
	}

	os.Stdout.Write(raw)
	return nil
}
