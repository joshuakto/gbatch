package gcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Executor runs gcloud commands and returns parsed JSON output.
// Production uses RealExecutor; tests use MockExecutor.
type Executor interface {
	// Run executes a gcloud command with --format=json and returns parsed output.
	Run(ctx context.Context, args ...string) (json.RawMessage, error)
	// RunRaw executes a gcloud command and returns raw stdout bytes.
	RunRaw(ctx context.Context, args ...string) ([]byte, error)
}

// RealExecutor shells out to the gcloud CLI.
type RealExecutor struct {
	// GcloudPath is the path to the gcloud binary. If empty, "gcloud" is used.
	GcloudPath string
}

// NewExecutor creates a RealExecutor, verifying gcloud is installed.
func NewExecutor() (*RealExecutor, error) {
	path, err := exec.LookPath("gcloud")
	if err != nil {
		return nil, fmt.Errorf("gcloud not found. Install: https://cloud.google.com/sdk/docs/install")
	}
	return &RealExecutor{GcloudPath: path}, nil
}

// Run executes gcloud with --format=json appended and parses the JSON output.
func (e *RealExecutor) Run(ctx context.Context, args ...string) (json.RawMessage, error) {
	args = append(args, "--format=json")
	out, err := e.RunRaw(ctx, args...)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(out) {
		return nil, fmt.Errorf("gcloud returned invalid JSON: %s", truncate(string(out), 200))
	}
	return json.RawMessage(out), nil
}

// RunRaw executes gcloud and returns raw stdout.
func (e *RealExecutor) RunRaw(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, e.GcloudPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, &GcloudError{
			Args:    args,
			Message: errMsg,
			Err:     err,
		}
	}
	return stdout.Bytes(), nil
}

// GcloudError wraps errors from gcloud invocations with context.
type GcloudError struct {
	Args    []string
	Message string
	Err     error
}

func (e *GcloudError) Error() string {
	return fmt.Sprintf("gcloud %s: %s", strings.Join(e.Args, " "), e.Message)
}

func (e *GcloudError) Unwrap() error {
	return e.Err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
