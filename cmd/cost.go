package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var (
	costToday bool
	costMonth bool
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show cost estimates for jobs",
	Long: `Show estimated costs for batch jobs.

Examples:
  gbatch cost             Cost for all recent jobs
  gbatch cost --today     Today's estimated spend
  gbatch cost --month     This month's estimated spend`,
	RunE: runCost,
}

func init() {
	costCmd.Flags().BoolVar(&costToday, "today", false, "Show today's costs only")
	costCmd.Flags().BoolVar(&costMonth, "month", false, "Show this month's costs")
	rootCmd.AddCommand(costCmd)
}

func runCost(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	output.Info("Calculating costs...")

	gcloudArgs := []string{"batch", "jobs", "list", "--location", cfg.Region}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	resp, err := executor.Run(context.Background(), gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Could not fetch jobs: %v", err),
			"Run 'gbatch doctor' to check your GCP setup",
		)
		return nil
	}

	var jobs []batchJob
	if err := json.Unmarshal(resp, &jobs); err != nil {
		output.Error(fmt.Sprintf("Failed to parse jobs: %v", err))
		return nil
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs found. Submit one with: gbatch submit job.sh")
		return nil
	}

	// Filter by time period
	now := time.Now()
	var filtered []batchJob
	for _, j := range jobs {
		created, err := time.Parse(time.RFC3339, j.CreateTime)
		if err != nil {
			filtered = append(filtered, j) // include if can't parse
			continue
		}
		if costToday && created.YearDay() != now.YearDay() {
			continue
		}
		if costMonth && (created.Month() != now.Month() || created.Year() != now.Year()) {
			continue
		}
		filtered = append(filtered, j)
	}

	if len(filtered) == 0 {
		period := "period"
		if costToday {
			period = "today"
		} else if costMonth {
			period = "this month"
		}
		fmt.Printf("No jobs found for %s.\n", period)
		return nil
	}

	if jsonOutput {
		return output.JSON(filtered)
	}

	// Calculate and display costs
	headers := []string{"JOB ID", "STATUS", "RESOURCES", "EST. COST"}
	widths := []int{12, 10, 16, 10}
	var rows [][]string
	var totalCost float64

	for _, j := range filtered {
		jobID := extractJobID(j.Name)
		state := j.Status.State
		cpus := 0
		memMB := 0
		if len(j.TaskGroups) > 0 {
			cpus = j.TaskGroups[0].TaskSpec.ComputeResource.CPUMilli / 1000
			memMB = j.TaskGroups[0].TaskSpec.ComputeResource.MemoryMiB
		}

		mem := fmt.Sprintf("%dG", memMB/1024)
		machineType := guessMachineType(cpus, mem)
		hourlyRate := estimateHourlyCost(machineType, false)

		// Estimate runtime from create/update time
		hours := estimateRuntimeHours(j.CreateTime, j.UpdateTime, state)
		cost := hourlyRate * hours
		totalCost += cost

		rows = append(rows, []string{
			output.Blue_(jobID),
			output.StatusColor(state),
			fmt.Sprintf("%d CPU / %s", cpus, mem),
			fmt.Sprintf("$%.2f", cost),
		})
	}

	output.Table(headers, rows, widths)

	period := ""
	if costToday {
		period = " today"
	} else if costMonth {
		period = " this month"
	}
	fmt.Printf("\n%s\n", output.Bold_(fmt.Sprintf("Total estimated cost%s: $%.2f", period, totalCost)))
	output.Warn("Costs are estimates based on machine type × runtime. Actual billing may differ.")

	return nil
}

func estimateRuntimeHours(createTime, updateTime, state string) float64 {
	created, err1 := time.Parse(time.RFC3339, createTime)
	updated, err2 := time.Parse(time.RFC3339, updateTime)

	if err1 != nil || err2 != nil {
		return 1.0 // default 1 hour
	}

	switch strings.ToUpper(state) {
	case "SUCCEEDED", "FAILED":
		return updated.Sub(created).Hours()
	case "RUNNING":
		return time.Since(created).Hours()
	default:
		return 0
	}
}
