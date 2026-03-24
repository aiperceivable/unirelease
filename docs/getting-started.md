# Getting Started

This guide walks you through installing unirelease, running your first release, and customizing the pipeline for your project.

## Installation

Choose one:

```bash
# One-line installer (auto-detects OS/arch, installs to /usr/local/bin)
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash

# Via Go
go install github.com/aiperceivable/unirelease@latest

# Or download a binary from GitHub Releases
# https://github.com/aiperceivable/unirelease/releases
```

Verify:

```bash
unirelease --version
```

## Your First Release (Dry Run)

Start with `--dry-run` to preview what unirelease would do — nothing is executed.

```bash
cd /path/to/your/project
unirelease --dry-run
```

Sample output:

```
 [1/11] Detect project type
   [dry-run] Detected python (from pyproject.toml)
 [2/11] Read version
   [dry-run] Version 0.3.1, Tag v0.3.1
 [3/11] Verify environment
   [dry-run] Would verify: required tools for python
 ...
```

If detection is wrong, override with `--type`:

```bash
unirelease --dry-run --type node
```

## Running a Real Release

When the dry-run looks correct:

```bash
unirelease
```

Destructive steps (git_tag, github_release, publish) will prompt for confirmation:

```
About to Create git tag. Continue? [y/N]
```

For CI/CD, skip all prompts with `--yes`:

```bash
unirelease --yes
```

## Language Examples

### Python

Prerequisites: `python3`, `build`, `twine`, `pytest`

```bash
# Project structure
myproject/
  pyproject.toml    # must have [project].version
  src/
  tests/

# Release
cd myproject
unirelease
```

Pipeline runs: detect → read version from `pyproject.toml` → verify python/twine/build installed → check git → clean dist/ build/ *.egg-info → `python -m build` → `pytest` → `twine check dist/*` → git tag → GitHub Release → `twine upload`.

### Go

Prerequisites: `go`

```bash
# Project structure
myproject/
  go.mod
  VERSION           # contains "1.2.3"
  main.go

# Release
cd myproject
unirelease
```

Pipeline runs: detect → read version from `VERSION` file → verify go installed → check git → `go clean` → `go build -trimpath -ldflags ...` → `go test ./...` → `go vet ./...` → git tag → GitHub Release (uploads cross-compiled binaries) → publish is a no-op (Go uses git tags).

### Node.js

Prerequisites: `node`, and one of `pnpm`/`npm`/`yarn`/`bun`

```bash
# Project structure
myproject/
  package.json      # must have "version" field
  pnpm-lock.yaml    # lockfile determines package manager
  src/

# Release
cd myproject
unirelease
```

Package manager is auto-detected from lockfiles: `pnpm-lock.yaml` > `bun.lockb` > `yarn.lock` > `package-lock.json`.

### Rust

Prerequisites: `cargo`

```bash
# Project structure
myproject/
  Cargo.toml        # [package] version = "1.0.0"
  src/

# Release
cd myproject
unirelease
```

Pipeline runs: detect → read version from `Cargo.toml` → verify cargo → check git → `cargo clean` → `cargo build --release` → `cargo test` → `cargo package --list` → git tag → GitHub Release → `cargo publish`.

## Configuration

Create `.unirelease.toml` in your project root. This file is **entirely optional**.

### Skip steps you don't need

```toml
skip = ["clean", "test"]
```

Or from the CLI:

```bash
unirelease --skip clean,test
```

Both sources are merged — CLI skip adds to config skip.

### Custom commands

```toml
[commands]
build = "make release"
test = "make test-all"
clean = "make clean"
```

### Lifecycle hooks

```toml
[hooks]
pre_build = "echo 'generating assets...'"
post_build = "cp dist/bin /opt/staging/"
pre_publish = "npm run prepublish-checks"
post_publish = "curl -X POST https://hooks.example.com/released"
```

### Custom tag prefix

```toml
# Default: "v" → "v1.0.0"
tag_prefix = "v"

# Monorepo: "api/v" → "api/v1.0.0"
tag_prefix = "api/v"
```

## GitHub Token Setup

The `github_release` step needs a GitHub token. unirelease checks these sources in order:

1. `GITHUB_TOKEN` environment variable
2. `GH_TOKEN` environment variable
3. `gh auth token` (GitHub CLI — run `gh auth login` once)
4. `git config github.token`

If no token is found, the step is **skipped** (not an error).

For CI/CD, set the token as a secret:

```yaml
# .github/workflows/release.yml
- name: Release
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: unirelease --yes
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Install unirelease
        run: go install github.com/aiperceivable/unirelease@latest

      - name: Release
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: unirelease --yes --skip git_tag
```

Note: `--skip git_tag` because the tag already exists (it triggered the workflow).

### Generic CI

```bash
# Install
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash

# Release (non-interactive)
GITHUB_TOKEN="$TOKEN" unirelease --yes
```

## Common Scenarios

### Run only one step

```bash
unirelease --step build       # just build
unirelease --step test        # just test
unirelease --step git_tag     # just tag
```

### Override version

```bash
unirelease --set-version 2.0.0
```

This overrides the version read from the manifest file.

### Monorepo with multiple projects

```bash
# Release the api/ subdirectory
unirelease api/

# With custom tag prefix to avoid conflicts
# In api/.unirelease.toml:
#   tag_prefix = "api/v"
```

### See what each step does

```bash
unirelease --list-steps
```

This shows detailed per-step documentation including the exact commands run for each language.

## Troubleshooting

### "missing required tools: twine"

Install the missing tool. For Python:

```bash
pip install twine build
```

### "no distribution files in dist/"

The build step didn't produce artifacts. Run `unirelease --step build` to debug.

### "aborted: uncommitted changes"

Commit or stash your changes first, or use `--yes` to continue anyway.

### GitHub Release step is skipped

No GitHub token found. See [GitHub Token Setup](#github-token-setup).

### Wrong project type detected

Use `--type` to override:

```bash
unirelease --type python
```

Or set it permanently in `.unirelease.toml`:

```toml
type = "python"
```
