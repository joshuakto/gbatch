# Release Process

## Distribution Channels

gBatch is distributed through three channels:

| Channel | Command | Audience | Auto-updated? |
|---------|---------|----------|---------------|
| Homebrew | `brew install joshuakto/tap/gbatch` | macOS/Linux users | Yes (brew upgrade) |
| GitHub Releases | Download binary from releases page | All platforms | Manual |
| Go install | `go install github.com/joshuakto/gbatch@latest` | Go developers | Manual |

## How to Release

### 1. Prepare

```bash
# Ensure main is clean and tests pass
git checkout main
git pull
make check
```

### 2. Tag

```bash
# Determine version (check git log for what changed)
git tag v0.1.0
git push origin v0.1.0
```

### 3. Automated Release (CI)

Pushing a `v*` tag triggers the GitHub Actions release workflow:

1. Runs full test suite
2. Builds binaries for linux/darwin × amd64/arm64 (CGO_ENABLED=0)
3. Creates GitHub Release with changelog and binaries
4. Generates SHA256 checksums
5. Updates Homebrew tap formula

### 4. Manual Release (if CI isn't set up yet)

```bash
# Install goreleaser: brew install goreleaser
make snapshot    # dry run — builds but doesn't publish
make release     # builds and publishes to GitHub Releases
```

## Homebrew Tap Setup (one-time)

1. Create a repo: `github.com/joshuakto/homebrew-tap`
2. Create a GitHub Personal Access Token with `repo` scope
3. Add it as `HOMEBREW_TAP_GITHUB_TOKEN` in your repo's GitHub Secrets
4. goreleaser will auto-create `Formula/gbatch.rb` on each release

Users then install with:
```bash
brew tap joshuakto/tap
brew install gbatch
```

## Verifying a Release

After release, verify on a clean machine:

```bash
# Homebrew
brew tap joshuakto/tap
brew install gbatch
gbatch --version
gbatch doctor

# GitHub Release
curl -L https://github.com/joshuakto/gbatch/releases/latest/download/gbatch_$(uname -s)_$(uname -m).tar.gz | tar xz
./gbatch --version

# Go install
go install github.com/joshuakto/gbatch@latest
gbatch --version
```

## Rollback

If a release has a critical bug:

```bash
# Delete the GitHub release and tag
gh release delete v0.1.0 --yes
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# Fix, retag, re-release
git commit -m "fix: critical bug"
git tag v0.1.1
git push origin v0.1.1
```

Homebrew users: `brew upgrade gbatch` picks up the new version.
Go install users: `go install github.com/joshuakto/gbatch@latest` picks up latest.
Binary users: must re-download.
