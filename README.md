# gBatch

A lightweight job scheduler CLI for Google Cloud, replacing UGER with a familiar experience.

```bash
gbatch submit --cpus 8 --mem 32G --mount gs://my-data:/data align.sh
```

## Why gBatch?

| | gcloud batch | gBatch |
|---|---|---|
| Submit a job | 30-line JSON config | `gbatch submit --cpus 8 --mem 32G job.sh` |
| Cost per job | Not available | Shown on every job |
| Monthly spend | Billing Console | `gbatch cost --month` |
| Diagnose issues | Manual checking | `gbatch doctor` |
| Migrate from UGER | Rewrite all scripts | `gbatch migrate script.sh` |
| Interactive session | Manual VM lifecycle | `gbatch ish --cpus 8 --mem 32G` |

## Install

```bash
brew install joshuakto/tap/gbatch
# or
go install github.com/joshuakto/gbatch@latest
```

## Commands

```
gbatch submit [script] [flags]     Submit a job (supports --mount)
gbatch status [job-id]             Colored job table with cost
gbatch logs [job-id]               Stream job logs
gbatch cancel [job-id]             Cancel a running job
gbatch cost [flags]                Cost reporting (--today, --month)
gbatch retry [job-id] [flags]      Retry with modified resources (--mem 2x)
gbatch migrate [script|--dir]      Convert UGER qsub scripts
gbatch ish [flags]                 Interactive shell (qlogin equivalent)
gbatch doctor                      Check GCP setup and permissions
gbatch config [key] [value]        Manage .gbatchrc defaults
gbatch completion [shell]          Generate shell completions
```

## Requirements

- [Google Cloud SDK](https://cloud.google.com/sdk) (`gcloud` CLI installed and authenticated)
- A GCP project with the Batch API enabled

## Configuration

Create a `.gbatchrc` in your project directory or `~/.gbatch/config.yaml` for user defaults:

```yaml
project: my-gcp-project
region: us-central1
default_cpus: 4
default_mem: 16G
mounts:
  - gs://my-data:/data
  - gs://my-refs:/refs
```

## Documentation

- [Team Onboarding & Secure Setup](docs/ONBOARDING.md) — GCP permissions, IAM roles, admin setup, user onboarding
- [Development Guide](docs/DEVELOPMENT.md) — Build, test, contribute, project structure
- [Release Process](docs/RELEASING.md) — Versioning, distribution channels, Homebrew tap
- [CI/CD Setup](docs/CICD_SETUP.md) — GitHub Actions pipeline, token setup, Go proxy, supply chain security
- [Design System](DESIGN.md) — CLI output patterns, colors, accessibility

## License

MIT
