# gBatch

## Overview
Lightweight Go CLI (~750 LOC) that wraps `gcloud` to provide a better job submission experience for teams migrating from UGER to Google Cloud Batch.

## Architecture
- **Shells out to `gcloud`** — does NOT use the GCP Go SDK. All GCP calls go through `exec.Command("gcloud", ...)` with `--format=json`.
- **Dependencies:** cobra (CLI framework) + go-yaml (config parsing). That's it.
- **No server component.** Pure CLI tool.

## Key Patterns
- `internal/gcloud/exec.go` — shared `gcloud.Run()` helper. All commands use this. Never shell out to gcloud directly from cmd/ files.
- `internal/output/output.go` — table/JSON/color rendering. Supports `NO_COLOR`, `TERM=dumb`, and `-o json`.
- `internal/config/config.go` — loads `.gbatchrc` (project) > `~/.gbatch/config.yaml` (user) > defaults. CLI flags override everything.
- All gcloud calls use `--format=json` for stable output parsing.

## Design System
Always read DESIGN.md before making any visual or UI decisions.
All font choices, colors, spacing, and aesthetic direction are defined there.
Do not deviate without explicit user approval.

## Testing
- `internal/gcloud/exec.go` exposes an `Executor` interface for mocking.
- Unit tests use mock executor with canned JSON responses.
- Golden files in `testdata/` validate parsing against real gcloud output.
- UGER migration parser should be fuzz-tested.
