# CI/CD Setup Guide

## What's Already Configured

| File | Purpose | Status |
|------|---------|--------|
| `.github/workflows/ci.yaml` | Test on push/PR, release on `v*` tag | Ready — works once repo is on GitHub |
| `.goreleaser.yaml` | Cross-compile binaries, publish to GitHub Releases + Homebrew | Ready — works once secrets are set |
| `Makefile` | Local build/test/release commands | Ready |

## One-Time Setup Checklist

### Step 1: Create GitHub Repo

```bash
cd /Users/joshuaching/Lucere/gbatch

# Option A: Public repo (anyone can install)
gh repo create joshuakto/gbatch --public --source=. --push

# Option B: Private repo (team only — Homebrew tap won't work for external users)
gh repo create joshuakto/gbatch --private --source=. --push
```

### Step 2: Create Homebrew Tap Repo

The Homebrew tap is a separate GitHub repo that goreleaser pushes formula updates to.

```bash
# Create the tap repo
gh repo create joshuakto/homebrew-tap --public --description "Homebrew formulae for joshuakto tools"

# Clone and add a README
gh repo clone joshuakto/homebrew-tap /tmp/homebrew-tap
echo "# joshuakto Homebrew Tap\n\n\`\`\`bash\nbrew tap joshuakto/tap\nbrew install gbatch\n\`\`\`" > /tmp/homebrew-tap/README.md
cd /tmp/homebrew-tap && git add . && git commit -m "init" && git push
```

After the first gBatch release, goreleaser will auto-create `Formula/gbatch.rb` in this repo.

### Step 3: Create GitHub Personal Access Token for Homebrew

goreleaser needs a token with permission to push to the `homebrew-tap` repo.

1. Go to https://github.com/settings/tokens?type=beta (Fine-grained tokens)
2. Create a new token:
   - **Name:** `gbatch-homebrew-tap`
   - **Expiration:** 1 year (set a calendar reminder to rotate)
   - **Repository access:** Select `joshuakto/homebrew-tap` only
   - **Permissions:** Contents → Read and write
3. Copy the token

### Step 4: Add Secret to gBatch Repo

```bash
# Add the Homebrew tap token as a repo secret
gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo joshuakto/gbatch
# Paste the token when prompted
```

`GITHUB_TOKEN` is automatically provided by GitHub Actions — no setup needed.

### Step 5: Verify CI Works

```bash
# Push to trigger CI
git push origin main

# Check CI status
gh run list --repo joshuakto/gbatch
```

### Step 6: First Release

```bash
# Tag and push
git tag v0.1.0
git push origin v0.1.0

# Watch the release
gh run watch --repo joshuakto/gbatch

# Verify
gh release view v0.1.0 --repo joshuakto/gbatch
```

After release completes, verify all channels:

```bash
# GitHub Releases — binaries should be attached
gh release view v0.1.0

# Homebrew — formula should exist in tap repo
gh api repos/joshuakto/homebrew-tap/contents/Formula/gbatch.rb --jq .name

# Go proxy — may take a few minutes to index
# Users can then: go install github.com/joshuakto/gbatch@v0.1.0
```

## How the Pipeline Works

```
DEVELOPER WORKFLOW:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  git push (branch)
       │
       ▼
  ┌─────────────────────────────────────────────────┐
  │ CI: Test Job (every push + PR)                   │
  │                                                  │
  │  1. go vet ./...                                 │
  │  2. go test ./... -v -coverprofile=coverage.out  │
  │  3. go build -o gbatch .                         │
  │  4. ./gbatch --help  (smoke test)                │
  │                                                  │
  │  Result: ✓ pass → PR is safe to merge            │
  │          ✗ fail → PR blocked                     │
  └─────────────────────────────────────────────────┘

  git tag v0.1.0 && git push origin v0.1.0
       │
       ▼
  ┌─────────────────────────────────────────────────┐
  │ CD: Release Job (only on v* tags)                │
  │                                                  │
  │  1. Run tests (same as CI)                       │
  │  2. goreleaser builds:                           │
  │     • linux/amd64, linux/arm64                   │
  │     • darwin/amd64, darwin/arm64                  │
  │     All with CGO_ENABLED=0 (static binaries)     │
  │  3. Creates GitHub Release with:                 │
  │     • 4 platform binaries (.tar.gz)              │
  │     • SHA256 checksums.txt                       │
  │     • Auto-generated changelog from commits      │
  │  4. Pushes Formula/gbatch.rb to homebrew-tap     │
  │                                                  │
  └─────────────────────────────────────────────────┘
       │
       ▼
  ┌─────────────────────────────────────────────────┐
  │ DISTRIBUTION (automatic after release)           │
  │                                                  │
  │  GitHub Releases: binaries downloadable          │
  │  Homebrew: brew upgrade gbatch picks up new ver  │
  │  Go proxy: go install ...@latest picks up tag    │
  │            (indexed by proxy.golang.org in ~5min) │
  └─────────────────────────────────────────────────┘

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Go Module Proxy (no CI needed)

Go modules are distributed via `proxy.golang.org` — there's no registry to push to. When you push a git tag like `v0.1.0` to a **public** GitHub repo, the Go proxy automatically indexes it within minutes. Users can then run:

```bash
go install github.com/joshuakto/gbatch@v0.1.0
go install github.com/joshuakto/gbatch@latest
```

For **private** repos, users need `GOPRIVATE=github.com/joshuakto/*` in their environment and git credentials configured. The Go proxy won't index private repos.

## Homebrew Tap (goreleaser handles it)

goreleaser auto-generates and pushes a Homebrew formula on every release. The formula includes:
- Binary download URLs for each platform
- SHA256 checksums for integrity verification
- Version pinning

Users install once:
```bash
brew tap joshuakto/tap
brew install gbatch
```

Upgrades are automatic:
```bash
brew upgrade gbatch
```

## Security Notes

### Token Rotation

The `HOMEBREW_TAP_GITHUB_TOKEN` should be rotated annually:
1. Create new token (Step 3 above)
2. Update secret: `gh secret set HOMEBREW_TAP_GITHUB_TOKEN --repo joshuakto/gbatch`
3. Delete old token in GitHub settings

### Binary Integrity

goreleaser generates `checksums.txt` with SHA256 hashes for every binary. Users can verify:

```bash
sha256sum -c checksums.txt
```

### Supply Chain

- CI builds from source on every release (no pre-built binaries checked in)
- `CGO_ENABLED=0` means fully static binaries — no dynamic linking, no system library dependencies
- Dependencies are pinned in `go.sum` with cryptographic hashes
- Homebrew formula includes SHA256 of the archive — tampered downloads fail install

### Signing (optional, not configured)

For additional trust, you can add GPG or cosign signing:

```yaml
# Add to .goreleaser.yaml:
signs:
  - artifacts: checksum
    args: ["--batch", "-u", "YOUR-GPG-KEY-ID", "--output", "${signature}", "--detach-sign", "${artifact}"]
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| CI doesn't trigger | Check `.github/workflows/ci.yaml` is on `main` branch |
| Release job skipped | Tag must start with `v` (e.g., `v0.1.0`, not `0.1.0`) |
| goreleaser fails "token" | `HOMEBREW_TAP_GITHUB_TOKEN` secret not set or expired |
| Homebrew formula not updated | Check `joshuakto/homebrew-tap` repo for push errors |
| `go install` returns old version | Go proxy caches — wait 5 min, or use `GONOSUMCHECK` |
| Private repo can't be `go install`'d | Set `GOPRIVATE=github.com/joshuakto/*` |
