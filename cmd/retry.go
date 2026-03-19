package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var retryMem string

var retryCmd = &cobra.Command{
	Use:   "retry [job-id]",
	Short: "Retry a failed job with modified resources",
	Long: `Retry a failed job, optionally with increased resources.

Examples:
  gbatch retry j-4819                  Retry with same resources
  gbatch retry j-4819 --mem 128G       Retry with 128G memory
  gbatch retry j-4819 --mem 2x         Retry with double memory`,
	Args: cobra.ExactArgs(1),
	RunE: runRetry,
}

func init() {
	retryCmd.Flags().StringVar(&retryMem, "mem", "", "Override memory (e.g., 128G or 2x)")
	rootCmd.AddCommand(retryCmd)
}

func runRetry(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	jobID := args[0]
	ctx := context.Background()

	// Fetch original job
	output.Info(fmt.Sprintf("Fetching original job %s...", jobID))
	gcloudArgs := []string{"batch", "jobs", "describe", jobID, "--location", cfg.Region}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	resp, err := executor.Run(ctx, gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Job not found: %s", jobID),
			"Job may have expired (GCP retains jobs for 60 days)",
		)
		return nil
	}

	var job batchJob
	if err := json.Unmarshal(resp, &job); err != nil {
		output.Error(fmt.Sprintf("Failed to parse job: %v", err))
		return nil
	}

	if len(job.TaskGroups) == 0 {
		output.Error("Could not read resource spec from original job")
		return nil
	}

	// Extract original resources
	origCPUs := job.TaskGroups[0].TaskSpec.ComputeResource.CPUMilli / 1000
	origMemMB := job.TaskGroups[0].TaskSpec.ComputeResource.MemoryMiB

	// Apply memory override
	newMemMB := origMemMB
	if retryMem != "" {
		if strings.HasSuffix(retryMem, "x") {
			// Multiplier: --mem 2x
			multiplierStr := strings.TrimSuffix(retryMem, "x")
			multiplier, err := strconv.ParseFloat(multiplierStr, 64)
			if err != nil {
				output.ErrorHint(
					fmt.Sprintf("Invalid memory multiplier: %s", retryMem),
					"Use format like '2x' or '128G'",
				)
				return nil
			}
			newMemMB = int(float64(origMemMB) * multiplier)
		} else {
			newMemMB = parseMemToMB(retryMem)
		}
	}

	origMemStr := fmt.Sprintf("%dG", origMemMB/1024)
	newMemStr := fmt.Sprintf("%dG", newMemMB/1024)

	output.Info(fmt.Sprintf("Resubmitting with %d CPU / %s...", origCPUs, newMemStr))

	// Build and submit new job
	jobConfig := buildJobConfig("", origCPUs, newMemStr, 0, "", nil, false)

	tmpFile, err := writeTempJSON(jobConfig)
	if err != nil {
		output.Error(fmt.Sprintf("Failed to create temp file: %v", err))
		return nil
	}

	submitArgs := []string{
		"batch", "jobs", "submit",
		"--location", cfg.Region,
		"--config", tmpFile,
	}
	if cfg.Project != "" {
		submitArgs = append(submitArgs, "--project", cfg.Project)
	}

	submitResp, err := executor.Run(ctx, submitArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Retry failed: %v", err),
			"Run 'gbatch doctor' to check your GCP setup",
		)
		return nil
	}

	if jsonOutput {
		fmt.Println(string(submitResp))
		return nil
	}

	// Parse new job ID
	var result map[string]any
	if err := json.Unmarshal(submitResp, &result); err == nil {
		if name, ok := result["name"].(string); ok {
			parts := strings.Split(name, "/")
			newJobID := parts[len(parts)-1]
			if origMemStr != newMemStr {
				output.Success(fmt.Sprintf("Job %s submitted (retry of %s, %s → %s)", newJobID, jobID, origMemStr, newMemStr))
			} else {
				output.Success(fmt.Sprintf("Job %s submitted (retry of %s)", newJobID, jobID))
			}
		}
	} else {
		output.Success(fmt.Sprintf("Retry of %s submitted", jobID))
	}

	return nil
}

func writeTempJSON(v any) (string, error) {
	f, err := createTempFile("gbatch-retry-*.json")
	if err != nil {
		return "", err
	}
	if err := json.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		return "", err
	}
	name := f.Name()
	f.Close()
	return name, nil
}
