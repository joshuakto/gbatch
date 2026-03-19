package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/joshuakto/gbatch/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check GCP setup and permissions",
	Long: `Diagnose your gBatch environment: gcloud installation, authentication,
project configuration, API access, and orphaned VMs.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fmt.Println(output.Bold_("Checking gBatch environment...\n"))
	ctx := context.Background()
	allGood := true

	// 1. Check gcloud installed
	gcloudPath, err := exec.LookPath("gcloud")
	if err != nil {
		output.Error("GCP SDK (gcloud) not found")
		fmt.Println("  Install: https://cloud.google.com/sdk/docs/install")
		return nil
	}
	output.Success(fmt.Sprintf("GCP SDK found: %s", gcloudPath))

	// Create executor for remaining checks
	exec, err := initExecutorForDoctor()
	if err != nil {
		output.Error(fmt.Sprintf("Cannot initialize gcloud: %v", err))
		return nil
	}

	// 2. Check authentication
	authResp, err := exec.RunRaw(ctx, "auth", "print-access-token")
	if err != nil || len(strings.TrimSpace(string(authResp))) == 0 {
		output.Error("Not authenticated")
		fmt.Println("  Run: gcloud auth application-default login")
		allGood = false
	} else {
		output.Success("Authenticated")
	}

	// 3. Check project
	projResp, err := exec.RunRaw(ctx, "config", "get-value", "project")
	project := strings.TrimSpace(string(projResp))
	if err != nil || project == "" || project == "(unset)" {
		output.Error("No GCP project set")
		fmt.Println("  Run: gcloud config set project YOUR-PROJECT-ID")
		allGood = false
	} else {
		if cfg.Project != "" {
			project = cfg.Project
		}
		output.Success(fmt.Sprintf("Project: %s", project))
	}

	// 4. Check Batch API enabled
	_, err = exec.RunRaw(ctx, "services", "list", "--filter=NAME:batch.googleapis.com", "--format=value(NAME)")
	if err != nil {
		output.Warn("Cannot verify Batch API status")
		allGood = false
	} else {
		output.Success("Batch API available")
	}

	// 5. Check for orphaned ish VMs
	orphanResp, err := exec.RunRaw(ctx, "compute", "instances", "list",
		"--filter=name~gbatch-ish-", "--format=value(name,zone,creationTimestamp)")
	if err == nil {
		orphans := strings.TrimSpace(string(orphanResp))
		if orphans != "" {
			lines := strings.Split(orphans, "\n")
			output.Warn(fmt.Sprintf("Found %d orphaned VM(s):", len(lines)))
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
			fmt.Println("  Delete with: gcloud compute instances delete VM-NAME --zone ZONE --quiet")
			allGood = false
		} else {
			output.Success("No orphaned VMs")
		}
	}

	// 6. Config status
	if cfg.Project != "" || cfg.Region != "us-central1" {
		output.Success(fmt.Sprintf("Config: project=%s, region=%s", cfg.Project, cfg.Region))
	} else {
		output.Warn("No .gbatchrc found (using defaults)")
		fmt.Println("  Create .gbatchrc with: gbatch config project YOUR-PROJECT-ID")
	}

	fmt.Println()
	if allGood {
		output.Success("Ready to use gBatch!")
	} else {
		output.Warn("Some issues found. Fix the items above and run 'gbatch doctor' again.")
	}

	return nil
}

func initExecutorForDoctor() (*gcloudExecWrapper, error) {
	path, err := exec.LookPath("gcloud")
	if err != nil {
		return nil, err
	}
	return &gcloudExecWrapper{path: path}, nil
}

// gcloudExecWrapper is a minimal executor for doctor that doesn't use --format=json.
type gcloudExecWrapper struct {
	path string
}

func (w *gcloudExecWrapper) RunRaw(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, w.path, args...)
	out, err := cmd.Output()
	return out, err
}
