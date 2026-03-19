package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [job-id]",
	Short: "Show job status",
	Long: `List all jobs or show details for a specific job.

Examples:
  gbatch status             List all recent jobs
  gbatch status j-4821      Show details for a specific job`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type batchJob struct {
	Name   string `json:"name"`
	Status struct {
		State string `json:"state"`
	} `json:"status"`
	TaskGroups []struct {
		TaskSpec struct {
			ComputeResource struct {
				CPUMilli  int `json:"cpuMilli"`
				MemoryMiB int `json:"memoryMib"`
			} `json:"computeResource"`
		} `json:"taskSpec"`
	} `json:"taskGroups"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	ctx := context.Background()

	if len(args) == 1 {
		return showJobDetail(ctx, args[0])
	}
	return listJobs(ctx)
}

func listJobs(ctx context.Context) error {
	output.Info("Fetching jobs...")

	gcloudArgs := []string{"batch", "jobs", "list", "--location", cfg.Region}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	resp, err := executor.Run(ctx, gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Could not fetch jobs: %v", err),
			"Run 'gbatch doctor' to check your GCP setup",
		)
		return nil
	}

	var jobs []batchJob
	if err := json.Unmarshal(resp, &jobs); err != nil {
		output.Error(fmt.Sprintf("Failed to parse job list: %v", err))
		return nil
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs found. Submit one with: gbatch submit job.sh")
		return nil
	}

	if jsonOutput {
		return output.JSON(jobs)
	}

	// Render table
	headers := []string{"JOB ID", "STATUS", "CPUS", "MEM", "COST"}
	widths := []int{12, 10, 4, 6, 8}
	var rows [][]string

	var totalCost float64
	counts := map[string]int{}

	for _, j := range jobs {
		jobID := extractJobID(j.Name)
		state := j.Status.State
		counts[state]++

		cpus := 0
		memMB := 0
		if len(j.TaskGroups) > 0 {
			cpus = j.TaskGroups[0].TaskSpec.ComputeResource.CPUMilli / 1000
			memMB = j.TaskGroups[0].TaskSpec.ComputeResource.MemoryMiB
		}
		memStr := fmt.Sprintf("%dG", memMB/1024)

		cost := 0.0 // TODO: estimate from runtime
		totalCost += cost

		rows = append(rows, []string{
			output.Blue_(jobID),
			output.StatusColor(state),
			fmt.Sprintf("%d", cpus),
			memStr,
			fmt.Sprintf("$%.2f", cost),
		})
	}

	output.Table(headers, rows, widths)

	// Summary line
	var parts []string
	parts = append(parts, fmt.Sprintf("%d jobs", len(jobs)))
	for _, state := range []string{"SUCCEEDED", "FAILED", "RUNNING", "QUEUED"} {
		if n, ok := counts[state]; ok && n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, strings.ToLower(state)))
		}
	}
	fmt.Println(output.Dim_(strings.Join(parts, " | ")))

	return nil
}

func showJobDetail(ctx context.Context, jobID string) error {
	output.Info(fmt.Sprintf("Fetching job %s...", jobID))

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

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	var job batchJob
	if err := json.Unmarshal(resp, &job); err != nil {
		output.Error(fmt.Sprintf("Failed to parse job: %v", err))
		return nil
	}

	fmt.Printf("%s %s\n", output.Bold_("Job:"), output.Blue_(extractJobID(job.Name)))
	fmt.Printf("%s %s\n", output.Bold_("Status:"), output.StatusColor(job.Status.State))
	if len(job.TaskGroups) > 0 {
		res := job.TaskGroups[0].TaskSpec.ComputeResource
		fmt.Printf("%s %d CPU / %dG\n", output.Bold_("Resources:"), res.CPUMilli/1000, res.MemoryMiB/1024)
	}
	fmt.Printf("%s %s\n", output.Bold_("Created:"), job.CreateTime)

	return nil
}

func extractJobID(fullName string) string {
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}
