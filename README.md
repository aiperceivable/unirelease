# unirelease

Unified release pipeline for multi-language projects. One command to detect, build, test, tag, release, and publish — for Rust, Go, Node.js, Bun, and Python.

**Zero config required.** Drop into any project directory and run `unirelease`. It auto-detects the project type from manifest files and runs the right pipeline.

## Install

```bash
# One-line installer (auto-detects OS/arch)
curl -sSL https://raw.githubusercontent.com/aipartnerup/unirelease/main/scripts/install.sh | bash

# Or via Go
go install github.com/aipartnerup/unirelease@latest
```

Also available as pre-built binaries and Docker. See [docs/releasing.md](docs/releasing.md) for all methods + CI/CD integration.

## Quick Start

```bash
unirelease              # auto-detect + full pipeline
unirelease --dry-run    # preview without executing
unirelease --yes        # non-interactive (CI/CD)
```

## Supported Languages

| Language | Manifest | Build | Test | Publish | Registry |
|----------|----------|-------|------|---------|----------|
| **Rust** | `Cargo.toml` | `cargo build --release` | `cargo test` | `cargo publish` | crates.io |
| **Go** | `go.mod` + `VERSION` | `go build -trimpath -ldflags ...` | `go test ./...` | git tag (auto) | proxy.golang.org |
| **Node.js** | `package.json` | `pnpm build` / `npm run build` | `pnpm test` / `npm run test` | `npm publish` | npm |
| **Bun** | `package.json` + `bun build --compile` | `bun run build` | `bun test` | Binary upload | GitHub Release |
| **Python** | `pyproject.toml` | `python -m build` | `pytest` | `twine upload` | PyPI |

Node.js automatically detects your package manager from lockfiles (`pnpm-lock.yaml` > `bun.lockb` > `yarn.lock` > `package-lock.json`).

## Pipeline Steps

unirelease runs 11 steps in order:

```
 1. detect           Auto-detect project type from manifest files
 2. read_version     Read version from manifest (or VERSION file for Go)
 3. verify_env       Check that required tools are installed
 4. check_git_status Verify clean working tree, warn on uncommitted changes
 5. clean            Remove build artifacts (dist/, build/, target/, etc.)
 6. build            Build the project with language-specific tooling
 7. test             Run the test suite
 8. verify           Verify the package (npm pack --dry-run, twine check, go vet)
 9. git_tag          Create annotated git tag and push to remote
10. github_release   Create GitHub Release with notes from CHANGELOG.md
11. publish          Publish to package registry (with pre-check for existing versions)
```

Steps marked as **destructive** (git_tag, github_release, publish) prompt for confirmation before executing. Use `--yes` to skip prompts.

## CLI Flags

```
unirelease [path] [flags]

Flags:
  --dry-run          Preview pipeline without executing
  --step <name>      Run only a specific step
  --type <type>      Override auto-detection (rust|go|node|bun|python)
  -v, --version           Print version
  -V, --set-version <X.Y.Z>  Override detected version
  -y, --yes          Non-interactive mode (skip confirmations)
```

### Examples

```bash
# Preview what would happen
unirelease --dry-run

# Release a specific directory
unirelease /path/to/project

# Override detected version
unirelease --set-version 2.0.0

# Run only the build step
unirelease --step build

# Force project type (useful for monorepos)
unirelease --type rust

# CI/CD: fully automated, no prompts
unirelease --yes
```

## Configuration

Create `.unirelease.toml` in your project root to customize behavior. **This file is entirely optional** — unirelease works without it.

```toml
# Override auto-detected project type
type = "rust"

# Custom tag prefix (default: "v" → "v1.0.0")
# Use language prefix for monorepos: "rust/v" → "rust/v1.0.0"
tag_prefix = "v"

# Skip specific steps
skip = ["clean", "verify"]

# Custom commands (override provider defaults)
[commands]
build = "make release"
test = "make test"
clean = "make clean"

# Lifecycle hooks
[hooks]
pre_build = "echo 'Starting build...'"
post_build = "echo 'Build complete!'"
pre_publish = "echo 'Publishing...'"
post_publish = "echo 'Published!'"
```

### Configuration Reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | auto-detected | Project type override |
| `tag_prefix` | string | `"v"` | Prefix for git tags |
| `skip` | string[] | `[]` | Steps to skip |
| `commands.build` | string | provider default | Custom build command |
| `commands.test` | string | provider default | Custom test command |
| `commands.clean` | string | provider default | Custom clean command |
| `hooks.pre_build` | string | - | Run before build step |
| `hooks.post_build` | string | - | Run after build step |
| `hooks.pre_publish` | string | - | Run before publish step |
| `hooks.post_publish` | string | - | Run after publish step |

## Version Detection

Each language reads version from its standard manifest:

| Language | Source | Example |
|----------|--------|---------|
| Rust | `Cargo.toml` → `[package] version` | `version = "1.2.3"` |
| Go | `VERSION` file (or `--set-version` flag) | `1.2.3` |
| Node/Bun | `package.json` → `version` | `"version": "1.2.3"` |
| Python | `pyproject.toml` → `[project] version` | `version = "1.2.3"` |

Go projects use a `VERSION` file because `go.mod` doesn't contain a project version. A leading `v` prefix is automatically stripped (`v1.2.3` → `1.2.3`) to prevent double-prefix tags.

## CHANGELOG Support

If a `CHANGELOG.md` exists in the project root, unirelease extracts release notes for the current version and uses them as the GitHub Release body. Supported heading formats:

```markdown
## [1.2.3] - 2026-03-21    <!-- recommended -->
## [1.2.3]
## 1.2.3
## [v1.2.3]
## v1.2.3
```

Content between the matched heading and the next `## ` heading is extracted. Falls back to `"Release version X.Y.Z"` if no CHANGELOG or version section is found.

## GitHub Integration

### Authentication

unirelease resolves GitHub tokens in this priority order:

1. `GITHUB_TOKEN` environment variable
2. `GH_TOKEN` environment variable
3. `gh auth token` (GitHub CLI)
4. `git config github.token`

If no token is found, the `github_release` step is skipped with a warning (not an error).

### Release Assets

For project types that produce binaries (Go, Bun), built files from `dist/` are automatically uploaded as GitHub Release assets.

## Registry Pre-Check

Before publishing, unirelease checks if the version already exists on the target registry:

| Language | Check Method |
|----------|-------------|
| Rust | `https://crates.io/api/v1/crates/{name}/{version}` |
| Node | `npm view {name} versions --json` |
| Python | `pip index versions {name}` |
| Go/Bun | Skipped (published via git tag / GitHub Release) |

If the version exists, you'll be prompted to confirm or skip.

## Detection Priority

When multiple manifest files exist (e.g., monorepo with `go.mod` + `package.json`), the highest confidence wins:

| Type | Manifest | Confidence |
|------|----------|-----------|
| Rust | `Cargo.toml` | 100 |
| Go | `go.mod` | 95 |
| Python | `pyproject.toml` | 90 |
| Bun | `package.json` + `bun build --compile` | 80 |
| Node | `package.json` | 50 |

Use `--type` to override when auto-detection picks the wrong one.

## Summary Report

After the pipeline completes, unirelease displays a summary with:

- Step-by-step results (done / skipped / dry-run)
- Remote status checks: git tag on remote, GitHub Release exists, registry version exists

```
╔══════════════════════════════════════════════════════════╗
║  Release Summary                                        ║
╚══════════════════════════════════════════════════════════╝

  Version:  1.0.0
  Tag:      v1.0.0
  Type:     go

  [done]    Detect project type
  [done]    Read version
  [done]    Verify environment
  [done]    Check git status
  [done]    Clean build artifacts
  [done]    Build project
  [done]    Run tests
  [skip]    Verify package (not applicable)
  [done]    Create git tag
  [done]    Create GitHub Release
  [skip]    Publish to registry (not applicable)

  Status:
    Git Tag:        yes
    GitHub Release: yes

  Release complete!
```

## Project Structure

```
unirelease/
├── cmd/root.go                    # CLI entry point (Cobra)
├── internal/
│   ├── changelog/                 # CHANGELOG.md parser
│   ├── config/                    # .unirelease.toml loader
│   ├── detector/                  # Project type & version detection
│   ├── git/                       # Git operations (tag, push, status)
│   ├── github/                    # GitHub API client (go-github)
│   ├── pipeline/
│   │   ├── engine.go              # Pipeline orchestrator
│   │   ├── context.go             # Provider interface & shared context
│   │   └── steps/                 # 11 pipeline step implementations
│   ├── providers/                 # Language-specific providers
│   │   ├── rust.go
│   │   ├── go.go
│   │   ├── node.go
│   │   ├── bun.go
│   │   └── python.go
│   ├── runner/                    # Command executor (dry-run aware)
│   └── ui/                        # Colored output & interactive prompts
├── e2e_test.go                    # End-to-end pipeline tests
├── go.mod
└── main.go
```

## Development

```bash
# Run all tests (183 tests across 10 packages)
go test ./...

# Run with verbose output
go test ./... -v

# Run only E2E tests
go test -run TestE2E -v

# Run only git integration tests
go test ./internal/git/... -run Integration -v

# Build
go build -o unirelease .
```

## License

Apache-2.0
