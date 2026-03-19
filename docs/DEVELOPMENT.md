# Development Guide

## Prerequisites

- Go 1.23+
- `gcloud` CLI (for integration testing only — unit tests don't require it)

## Quick Start

```bash
git clone https://github.com/joshuakto/gbatch.git
cd gbatch
make build        # builds ./gbatch binary
make test         # runs all tests
```

## Project Structure

```
gbatch/
├── main.go                       Entry point
├── cmd/                          Cobra commands (one file per command)
│   ├── root.go                   Root command, global flags, version
│   ├── submit.go                 Job submission with --mount, --spot
│   ├── status.go                 Job listing with colored table
│   ├── logs.go                   Log streaming from Cloud Logging
│   ├── cancel.go                 Job cancellation
│   ├── cost.go                   Cost estimation and reporting
│   ├── retry.go                  Retry with resource escalation
│   ├── migrate.go                UGER script converter
│   ├── ish.go                    Interactive shell session
│   ├── doctor.go                 Environment diagnostics
│   └── config_cmd.go             .gbatchrc management
├── internal/
│   ├── gcloud/
│   │   ├── exec.go               Executor interface + RealExecutor
│   │   └── mock.go               MockExecutor for testing
│   ├── config/config.go          Config loading with layered resolution
│   ├── output/output.go          Table, JSON, color, NO_COLOR support
│   └── migrate/parser.go         UGER #$ directive parser
├── testdata/                     Golden files for gcloud JSON responses
├── docs/                         Development and operations docs
├── DESIGN.md                     Visual design system
├── CLAUDE.md                     AI development instructions
├── Makefile                      Build, test, release commands
├── .goreleaser.yaml              Release automation config
└── .github/workflows/ci.yaml     CI/CD pipeline
```

## Key Architecture Rules

1. **Never import the GCP Go SDK.** All GCP calls go through `internal/gcloud/exec.go` which shells out to `gcloud` with `--format=json`.

2. **All commands use the `Executor` interface.** This makes every command unit-testable with `MockExecutor`.

3. **Config layering:** CLI flags > project `.gbatchrc` > user `~/.gbatch/config.yaml` > defaults.

4. **Output goes through `internal/output/`** — never use `fmt.Printf` with raw ANSI codes in `cmd/` files.

## Development Workflow

```bash
make build            # Build binary
make test             # Run all tests
make test-cover       # Tests with coverage report
make test-fuzz        # Fuzz test the UGER parser (30s)
make lint             # golangci-lint (install: brew install golangci-lint)
make vet              # go vet
make check            # vet + lint + test (full CI locally)
make fmt              # gofmt
```

## Adding a New Command

1. Create `cmd/mycommand.go`
2. Define cobra command with `Use`, `Short`, `Long`, `Args`, `RunE`
3. In `init()`, call `rootCmd.AddCommand(mycommandCmd)`
4. Use `initExecutor()` at the top of `RunE` if you need gcloud
5. Use `output.Success()`, `output.Error()`, `output.ErrorHint()` for user-facing messages
6. Support `-o json` by checking `jsonOutput` flag
7. Add tests with `MockExecutor`

## Testing

### Unit Tests (no gcloud required)

```go
func TestSubmit(t *testing.T) {
    mock := gcloud.NewMockExecutor()
    mock.Responses["batch"] = json.RawMessage(`{"name": "projects/p/locations/r/jobs/j-123"}`)
    executor = mock  // inject mock

    // run command, assert output
}
```

### Golden File Tests

Place real gcloud JSON responses in `testdata/`:
- `testdata/jobs_list.json` — response from `gcloud batch jobs list`
- `testdata/job_describe.json` — response from `gcloud batch jobs describe`

Tests parse these to validate the JSON→struct mapping matches real gcloud output.

### Fuzz Testing

The UGER migration parser handles untrusted input (user scripts). Fuzz it:

```bash
make test-fuzz
```

## Versioning

Follows [Semantic Versioning](https://semver.org/):
- **Patch** (v0.1.1): bug fixes, no behavior change
- **Minor** (v0.2.0): new commands or flags, backward-compatible
- **Major** (v1.0.0): breaking changes to CLI interface or config format

Version is injected at build time via ldflags:
```bash
go build -ldflags "-X github.com/joshuakto/gbatch/cmd.Version=v0.1.0" -o gbatch .
```
