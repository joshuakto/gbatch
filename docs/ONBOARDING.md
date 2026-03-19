# Team Onboarding & Secure Setup

This guide covers how to set up gBatch for a team migrating from UGER to GCP.

## Overview

```
SETUP FLOW:
┌──────────────────────┐     ┌──────────────────────┐     ┌──────────────────────┐
│ 1. ADMIN SETUP       │────▶│ 2. USER SETUP        │────▶│ 3. VERIFY            │
│ (once per org)       │     │ (once per user)       │     │                      │
│                      │     │                      │     │                      │
│ • Create GCP project │     │ • Install gcloud     │     │ • gbatch doctor      │
│ • Enable APIs        │     │ • Install gbatch     │     │ • gbatch submit test │
│ • Set IAM roles      │     │ • Authenticate       │     │                      │
│ • Create GCS buckets │     │ • Configure .gbatchrc │     │                      │
└──────────────────────┘     └──────────────────────┘     └──────────────────────┘
```

## 1. Admin Setup (once per organization)

### 1a. GCP Project Setup

```bash
# Create or select a project
gcloud projects create my-genomics-batch --name="Genomics Batch Computing"
gcloud config set project my-genomics-batch

# Link a billing account
gcloud billing accounts list
gcloud billing projects link my-genomics-batch --billing-account=BILLING-ACCOUNT-ID
```

### 1b. Enable Required APIs

```bash
gcloud services enable \
  batch.googleapis.com \
  compute.googleapis.com \
  logging.googleapis.com \
  storage.googleapis.com
```

### 1c. IAM Roles — Minimum Required Permissions

Grant each user the following roles on the project:

```bash
# For each user:
USER_EMAIL="researcher@org.com"

gcloud projects add-iam-policy-binding my-genomics-batch \
  --member="user:${USER_EMAIL}" \
  --role="roles/batch.jobsEditor"

gcloud projects add-iam-policy-binding my-genomics-batch \
  --member="user:${USER_EMAIL}" \
  --role="roles/logging.viewer"

gcloud projects add-iam-policy-binding my-genomics-batch \
  --member="user:${USER_EMAIL}" \
  --role="roles/compute.viewer"
```

#### Permission Matrix

| gBatch Feature | IAM Role | Level | Required? |
|----------------|----------|-------|-----------|
| Submit / cancel jobs | `roles/batch.jobsEditor` | Project | Yes |
| View job status | `roles/batch.viewer` | Project | Yes |
| View job logs | `roles/logging.viewer` | Project | Yes |
| Cost estimation | `roles/compute.viewer` | Project | Yes |
| Interactive session (ish) | `roles/compute.instanceAdmin.v1` | Project | For `ish` only |
| SSH into ish VMs | `roles/compute.osLogin` | Project | For `ish` only |
| Access GCS data (mounts) | `roles/storage.objectViewer` | Bucket | For `--mount` |
| Exact billing data | `roles/billing.viewer` | Billing Account | Optional |

#### For `gbatch ish` Users (additional)

```bash
gcloud projects add-iam-policy-binding my-genomics-batch \
  --member="user:${USER_EMAIL}" \
  --role="roles/compute.instanceAdmin.v1"

gcloud projects add-iam-policy-binding my-genomics-batch \
  --member="user:${USER_EMAIL}" \
  --role="roles/compute.osLogin"
```

#### Batch Admin Script

For onboarding many users, create `scripts/onboard-user.sh`:

```bash
#!/bin/bash
set -euo pipefail

PROJECT="my-genomics-batch"
USER_EMAIL="$1"

if [ -z "$USER_EMAIL" ]; then
  echo "Usage: ./onboard-user.sh user@org.com"
  exit 1
fi

ROLES=(
  "roles/batch.jobsEditor"
  "roles/logging.viewer"
  "roles/compute.viewer"
  "roles/compute.instanceAdmin.v1"
  "roles/compute.osLogin"
)

for ROLE in "${ROLES[@]}"; do
  echo "Granting ${ROLE}..."
  gcloud projects add-iam-policy-binding "$PROJECT" \
    --member="user:${USER_EMAIL}" \
    --role="$ROLE" \
    --quiet
done

echo "✓ ${USER_EMAIL} onboarded to ${PROJECT}"
```

### 1d. Data Buckets

```bash
# Create shared data buckets
gcloud storage buckets create gs://my-genomics-data --location=us-central1
gcloud storage buckets create gs://my-genomics-refs --location=us-central1

# Grant team read access
gcloud storage buckets add-iam-policy-binding gs://my-genomics-data \
  --member="group:genomics-team@org.com" \
  --role="roles/storage.objectViewer"
```

### 1e. Team-Wide Configuration (optional)

Create a shared `.gbatchrc` and commit it to your team's repo:

```yaml
project: my-genomics-batch
region: us-central1
default_cpus: 4
default_mem: 16G
mounts:
  - gs://my-genomics-data:/data
  - gs://my-genomics-refs:/refs
```

## 2. User Setup (each team member)

### 2a. Install gcloud

```bash
# macOS
brew install google-cloud-sdk

# Linux
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```

### 2b. Authenticate

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project my-genomics-batch
```

### 2c. Install gBatch

```bash
# macOS/Linux via Homebrew
brew install joshuakto/tap/gbatch

# Or download binary
curl -L https://github.com/joshuakto/gbatch/releases/latest/download/gbatch_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv gbatch /usr/local/bin/

# Or via Go
go install github.com/joshuakto/gbatch@latest
```

### 2d. Verify

```bash
gbatch doctor
```

Expected output:
```
Checking gBatch environment...

✓ GCP SDK found: /usr/local/bin/gcloud
✓ Authenticated
✓ Project: my-genomics-batch
✓ Batch API available
✓ No orphaned VMs
✓ Config: project=my-genomics-batch, region=us-central1

✓ Ready to use gBatch!
```

### 2e. Shell Completions (optional)

```bash
# bash
gbatch completion bash > /etc/bash_completion.d/gbatch

# zsh
gbatch completion zsh > "${fpath[1]}/_gbatch"

# fish
gbatch completion fish > ~/.config/fish/completions/gbatch.fish
```

## 3. Security Considerations

### Credentials

- gBatch **never stores credentials.** It uses GCP Application Default Credentials (ADC) via `gcloud auth`.
- `.gbatchrc` stores project ID and region only — never tokens, keys, or passwords.
- **Never commit service account keys** to the repo. Use `gcloud auth` for user access.

### Network Security

- All GCP API calls use HTTPS (enforced by `gcloud`).
- `gbatch ish` creates VMs in your VPC. They inherit your project's firewall rules.
- Spot/preemptible VMs used by `ish` auto-terminate after 4 hours max.

### Data Access

- `--mount` uses GCS FUSE, which respects IAM permissions on the bucket.
- Users can only mount buckets they have `storage.objectViewer` on.
- No data passes through gBatch — it's a direct GCS↔VM mount.

### Audit Trail

- All job submissions are logged in GCP Cloud Audit Logs.
- `gbatch ish` VMs are named `gbatch-ish-{user}-{timestamp}` for traceability.
- `gbatch doctor` detects orphaned VMs to prevent cost leaks.

### Principle of Least Privilege

Grant only the roles needed. The minimum for basic job submission is:
- `roles/batch.jobsEditor`
- `roles/logging.viewer`
- `roles/compute.viewer`

Add `compute.instanceAdmin` and `compute.osLogin` only for users who need `gbatch ish`.

## 4. Upgrading gBatch

```bash
# Homebrew
brew upgrade gbatch

# Go install
go install github.com/joshuakto/gbatch@latest

# Manual
curl -L https://github.com/joshuakto/gbatch/releases/latest/download/gbatch_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv gbatch /usr/local/bin/
```

Check version:
```bash
gbatch --version
```

## 5. Troubleshooting

| Problem | Diagnosis | Fix |
|---------|-----------|-----|
| "gcloud not found" | `gbatch doctor` | Install Google Cloud SDK |
| "Not authenticated" | `gbatch doctor` | `gcloud auth application-default login` |
| "Permission denied" on submit | Missing IAM role | Admin runs `onboard-user.sh` |
| "Permission denied" on logs | Missing logging.viewer | Admin grants `roles/logging.viewer` |
| "VM creation failed" on ish | Quota or missing role | Check quota; grant `compute.instanceAdmin` |
| Orphaned VMs after crash | `gbatch doctor` | Doctor detects and offers cleanup |
| Cost shows $0.00 | Job just started or < 1 min | Cost estimates use runtime; wait for job to run |
| Mount path not found in job | Wrong GCS path format | Use `gs://bucket-name:/mount-path` |
