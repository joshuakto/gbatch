package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var (
	ishCPUs   int
	ishMem    string
	ishMounts []string
)

var ishCmd = &cobra.Command{
	Use:   "ish",
	Short: "Start an interactive shell session (qlogin equivalent)",
	Long: `Create a VM with requested resources, SSH into it, and auto-delete on exit.

The VM is preemptible with a 4-hour max lifetime to prevent orphaned VMs.

Examples:
  gbatch ish --cpus 8 --mem 32G
  gbatch ish --cpus 4 --mem 16G --mount gs://my-data:/data`,
	RunE: runIsh,
}

func init() {
	ishCmd.Flags().IntVar(&ishCPUs, "cpus", 0, "Number of CPUs")
	ishCmd.Flags().StringVar(&ishMem, "mem", "", "Memory (e.g., 32G)")
	ishCmd.Flags().StringSliceVar(&ishMounts, "mount", nil, "GCS FUSE mount (gs://bucket:/path)")
	rootCmd.AddCommand(ishCmd)
}

func runIsh(cmd *cobra.Command, args []string) error {
	if err := initExecutor(); err != nil {
		output.Error(err.Error())
		return nil
	}

	cpus := ishCPUs
	if cpus == 0 {
		cpus = cfg.DefaultCPUs
	}
	mem := ishMem
	if mem == "" {
		mem = cfg.DefaultMem
	}
	mounts := ishMounts
	if len(mounts) == 0 {
		mounts = cfg.Mounts
	}

	machineType := guessMachineType(cpus, mem)
	vmName := fmt.Sprintf("gbatch-ish-%s-%s", os.Getenv("USER"), time.Now().Format("0102-1504"))
	region := cfg.Region
	zone := region + "-a" // default to zone a

	ctx := context.Background()
	startTime := time.Now()

	// Phase 1: Create VM
	output.Info(fmt.Sprintf("Creating VM... (%s, %s, preemptible)", zone, machineType))
	output.Warn("VM will auto-terminate after 4 hours")

	createArgs := []string{
		"compute", "instances", "create", vmName,
		"--zone", zone,
		"--machine-type", machineType,
		"--provisioning-model", "SPOT",
		"--max-run-duration", "4h",
		"--instance-termination-action", "DELETE",
		"--scopes", "cloud-platform",
	}
	if cfg.Project != "" {
		createArgs = append(createArgs, "--project", cfg.Project)
	}

	_, err := executor.Run(ctx, createArgs...)
	if err != nil {
		output.ErrorHint(
			fmt.Sprintf("VM creation failed: %v", err),
			"Check quota and permissions with 'gbatch doctor'",
		)
		return nil
	}

	// Set up cleanup on exit
	cleanup := func() {
		output.Info(fmt.Sprintf("Deleting VM %s...", vmName))
		deleteArgs := []string{
			"compute", "instances", "delete", vmName,
			"--zone", zone, "--quiet",
		}
		if cfg.Project != "" {
			deleteArgs = append(deleteArgs, "--project", cfg.Project)
		}
		_, err := executor.Run(context.Background(), deleteArgs...)
		if err != nil {
			output.Warn(fmt.Sprintf("Failed to delete VM: %v. Run 'gbatch doctor' to check for orphaned VMs.", err))
			return
		}
		duration := time.Since(startTime).Round(time.Minute)
		cost := estimateHourlyCost(machineType, true) * duration.Hours()
		output.Success(fmt.Sprintf("VM deleted. Session lasted %s. Est. cost: $%.2f", duration, cost))
	}

	// Trap signals for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()

	output.Success(fmt.Sprintf("VM %s ready", vmName))

	// Show mount info
	if len(mounts) > 0 {
		mountStrs := make([]string, len(mounts))
		for i, m := range mounts {
			parts := strings.SplitN(m, ":", 2)
			if len(parts) == 2 {
				mountStrs[i] = fmt.Sprintf("%s → %s", parts[1], parts[0])
			} else {
				mountStrs[i] = m
			}
		}
		fmt.Printf("  Mounts: %s\n", strings.Join(mountStrs, ", "))
	}

	// Phase 2: SSH into VM
	output.Info("Connecting via SSH...")
	fmt.Println(output.Dim_("Type 'exit' to end session and delete VM."))
	fmt.Println()

	sshArgs := []string{
		"compute", "ssh", vmName,
		"--zone", zone,
	}
	if cfg.Project != "" {
		sshArgs = append(sshArgs, "--project", cfg.Project)
	}

	// SSH needs to be interactive — use RunRaw and connect stdin/stdout
	_, err = executor.RunRaw(ctx, sshArgs...)
	if err != nil {
		output.Error(fmt.Sprintf("SSH connection failed: %v", err))
		cleanup()
		output.Warn("Hint: check your SSH keys with 'gcloud compute config-ssh'")
		return nil
	}

	// Phase 4: Cleanup after SSH exits
	cleanup()
	return nil
}
