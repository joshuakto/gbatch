package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var (
	submitCPUs   int
	submitMem    string
	submitGPU    int
	submitName   string
	submitMounts []string
	submitSpot   bool
)

var submitCmd = &cobra.Command{
	Use:   "submit [script]",
	Short: "Submit a job to Google Cloud Batch",
	Long: `Submit a batch job with resource specifications.

Examples:
  gbatch submit align.sh --cpus 8 --mem 32G
  gbatch submit --cpus 4 --mem 16G --mount gs://my-data:/data job.sh
  gbatch submit --spot --cpus 8 --mem 32G variant_call.sh`,
	Args: cobra.ExactArgs(1),
	RunE: runSubmit,
}

func init() {
	submitCmd.Flags().IntVar(&submitCPUs, "cpus", 0, "Number of CPUs")
	submitCmd.Flags().StringVar(&submitMem, "mem", "", "Memory (e.g., 32G)")
	submitCmd.Flags().IntVar(&submitGPU, "gpu", 0, "Number of GPUs")
	submitCmd.Flags().StringVar(&submitName, "name", "", "Job name")
	submitCmd.Flags().StringSliceVar(&submitMounts, "mount", nil, "GCS FUSE mount (gs://bucket:/path)")
	submitCmd.Flags().BoolVar(&submitSpot, "spot", false, "Use spot/preemptible VM")
	rootCmd.AddCommand(submitCmd)
}

func runSubmit(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	script := args[0]
	if _, err := os.Stat(script); os.IsNotExist(err) {
		output.ErrorHint(
			fmt.Sprintf("Script not found: %s", script),
			"Check the file path and try again",
		)
		return nil
	}

	// Apply config defaults
	cpus := submitCPUs
	if cpus == 0 {
		cpus = cfg.DefaultCPUs
	}
	mem := submitMem
	if mem == "" {
		mem = cfg.DefaultMem
	}
	mounts := submitMounts
	if len(mounts) == 0 {
		mounts = cfg.Mounts
	}

	// Build the gcloud batch job config JSON
	jobConfig := buildJobConfig(script, cpus, mem, submitGPU, submitName, mounts, submitSpot)

	// Write config to temp file
	tmpFile, err := os.CreateTemp("", "gbatch-job-*.json")
	if err != nil {
		output.Error(fmt.Sprintf("Failed to create temp file: %v", err))
		return nil
	}
	defer os.Remove(tmpFile.Name())

	if err := json.NewEncoder(tmpFile).Encode(jobConfig); err != nil {
		output.Error(fmt.Sprintf("Failed to write job config: %v", err))
		return nil
	}
	tmpFile.Close()

	// Show mount summary
	if len(mounts) > 0 {
		mountStrs := make([]string, len(mounts))
		for i, m := range mounts {
			parts := strings.SplitN(m, ":", 2)
			if len(parts) == 2 {
				mountStrs[i] = fmt.Sprintf("%s → %s", parts[0], parts[1])
			} else {
				mountStrs[i] = m
			}
		}
		output.Info(fmt.Sprintf("Mounts: %s", strings.Join(mountStrs, ", ")))
	}

	region := cfg.Region
	output.Info(fmt.Sprintf("Submitting to %s...", region))

	// Submit via gcloud
	gcloudArgs := []string{
		"batch", "jobs", "submit",
		"--location", region,
		"--config", tmpFile.Name(),
	}
	if cfg.Project != "" {
		gcloudArgs = append(gcloudArgs, "--project", cfg.Project)
	}

	resp, err := executor.Run(context.Background(), gcloudArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("Submit failed: %v", err),
			"Run 'gbatch doctor' to check your GCP setup",
		)
		return nil
	}

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	// Parse response for job ID
	var result map[string]any
	if err := json.Unmarshal(resp, &result); err == nil {
		if name, ok := result["name"].(string); ok {
			parts := strings.Split(name, "/")
			jobID := parts[len(parts)-1]
			output.Success(fmt.Sprintf("Job %s submitted", jobID))

			// Estimate cost
			machineType := guessMachineType(cpus, mem)
			output.Warn(fmt.Sprintf("Est. cost: ~$%.2f/hr (%s)", estimateHourlyCost(machineType, submitSpot), machineType))
		}
	} else {
		output.Success("Job submitted")
	}

	return nil
}

func buildJobConfig(script string, cpus int, mem string, gpu int, name string, mounts []string, spot bool) map[string]any {
	memMB := parseMemToMB(mem)

	taskSpec := map[string]any{
		"runnables": []map[string]any{
			{
				"script": map[string]any{
					"path": "/mnt/share/" + script,
				},
			},
		},
		"computeResource": map[string]any{
			"cpuMilli":  cpus * 1000,
			"memoryMib": memMB,
		},
	}

	// Add GCS FUSE mounts
	if len(mounts) > 0 {
		var volumes []map[string]any
		for _, m := range mounts {
			parts := strings.SplitN(m, ":", 2)
			if len(parts) != 2 {
				continue
			}
			bucket := strings.TrimPrefix(parts[0], "gs://")
			mountPath := parts[1]
			volumes = append(volumes, map[string]any{
				"gcs": map[string]any{
					"remotePath": bucket,
				},
				"mountPath": mountPath,
			})
		}
		taskSpec["volumes"] = volumes
	}

	config := map[string]any{
		"taskGroups": []map[string]any{
			{
				"taskSpec":  taskSpec,
				"taskCount": 1,
			},
		},
		"logsPolicy": map[string]any{
			"destination": "CLOUD_LOGGING",
		},
	}

	if spot {
		config["allocationPolicy"] = map[string]any{
			"instances": []map[string]any{
				{
					"policy": map[string]any{
						"provisioningModel": "SPOT",
					},
				},
			},
		}
	}

	return config
}

func parseMemToMB(mem string) int {
	mem = strings.ToUpper(strings.TrimSpace(mem))
	if strings.HasSuffix(mem, "G") {
		val := strings.TrimSuffix(mem, "G")
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n * 1024
	}
	if strings.HasSuffix(mem, "M") {
		val := strings.TrimSuffix(mem, "M")
		var n int
		fmt.Sscanf(val, "%d", &n)
		return n
	}
	return 16384 // default 16G
}

func guessMachineType(cpus int, mem string) string {
	memMB := parseMemToMB(mem)
	memGB := memMB / 1024
	ratio := 0
	if cpus > 0 {
		ratio = memGB / cpus
	}
	if ratio <= 4 {
		return fmt.Sprintf("n2-standard-%d", cpus)
	}
	return fmt.Sprintf("n2-highmem-%d", cpus)
}

func estimateHourlyCost(machineType string, spot bool) float64 {
	// Approximate GCP pricing (us-central1)
	costs := map[string]float64{
		"n2-standard-2":  0.097,
		"n2-standard-4":  0.194,
		"n2-standard-8":  0.388,
		"n2-standard-16": 0.776,
		"n2-standard-32": 1.552,
		"n2-highmem-2":   0.131,
		"n2-highmem-4":   0.262,
		"n2-highmem-8":   0.524,
		"n2-highmem-16":  1.048,
		"n2-highmem-32":  2.096,
	}
	cost, ok := costs[machineType]
	if !ok {
		cost = 0.388 // default to n2-standard-8
	}
	if spot {
		cost *= 0.3 // spot is ~60-80% cheaper
	}
	return cost
}
