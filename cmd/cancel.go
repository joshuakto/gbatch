package cmd

import (
	"context"
	"fmt"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel [job-id]",
	Short: "Cancel a running job",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancel,
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	jobID := args[0]
	output.Info(fmt.Sprintf("Cancelling %s...", jobID))

	gcloudArgs := []string{"batch", "jobs", "delete", jobID, "--location", cfg.Region}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	_, err := executor.Run(context.Background(), gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Cancel failed: %v", err),
			"Job may have already completed",
		)
		return nil
	}

	output.Success(fmt.Sprintf("Job %s cancelled", jobID))
	return nil
}
