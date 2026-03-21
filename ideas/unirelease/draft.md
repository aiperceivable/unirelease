# unirelease

> Status: Ready | Draft v1 | 2026-03-21

## One-Liner
A single Go binary that auto-detects project type (Rust, TypeScript/Bun, Node, Python) and runs a unified release pipeline, replacing per-language release scripts.

## Problem
Maintaining 3 separate release.sh scripts (TypeScript 543 lines, Rust 393 lines, Python ~500 lines) that all follow the same flow but are duplicated for each language. They share identical logic (git checks, tagging, GitHub releases) but diverge on build/publish commands. They don't work on Windows.

## Target Users
- Solo developers and small teams maintaining multi-language projects
- Monorepo maintainers who release packages in different ecosystems
- The immediate user: replacing 3 existing bash scripts in the apdev project

## Core Experience

### Auto-Detection
```
unirelease              # detect project type from cwd
unirelease /path/to/project  # explicit path
```

### Detection Rules
| File Present | Type | Build Command | Publish Target |
|-------------|------|---------------|----------------|
| Cargo.toml | Rust | `cargo build --release` | crates.io |
| package.json + `bun build --compile` in scripts | bun-binary | `bun run build` | GitHub Release (upload binary) |
| package.json (normal) | node | `pnpm build` | npm publish |
| pyproject.toml | Python | `python -m build` | PyPI |

### Unified Pipeline
```
detect -> read_version -> verify_env -> check_git_status -> clean -> build -> test -> git_tag -> github_release -> publish
```

Each step is extracted from the existing 3 scripts and unified into a single pipeline with language-specific implementations only where needed (build, publish).

## CLI Interface

```
unirelease                          # auto-detect and release
unirelease /path/to/project         # explicit project path
unirelease --step build             # run only a specific step
unirelease --yes                    # non-interactive mode (skip confirmations)
unirelease --dry-run                # preview without executing
unirelease --version 1.2.3          # override detected version
unirelease --type rust              # override auto-detection
```

### Flags Summary
| Flag | Short | Description |
|------|-------|-------------|
| `--step <step>` | | Run only a specific pipeline step |
| `--yes` | `-y` | Non-interactive mode |
| `--dry-run` | | Preview without executing |
| `--version <ver>` | | Override detected version |
| `--type <type>` | | Override auto-detection |

## Architecture

```
unirelease (Go binary)

+--------------------------------------------------+
|  CLI Layer (cobra)                                |
|  Parse flags, resolve project path                |
+--------------------------------------------------+
                     |
+--------------------------------------------------+
|  Detector                                        |
|  Scan files -> determine project type            |
|  Read version from: package.json / Cargo.toml /  |
|                     pyproject.toml               |
+--------------------------------------------------+
                     |
+--------------------------------------------------+
|  Pipeline Engine                                 |
|  Orchestrate steps, handle --step / --dry-run    |
+--------------------------------------------------+
                     |
+------+------+------+------+
| Rust | Bun  | Node | Py   |  <- Language-specific
| Impl | Impl | Impl | Impl |     build/publish
+------+------+------+------+
                     |
+--------------------------------------------------+
|  Shared Steps                                    |
|  verify_env, check_git_status, clean,            |
|  git_tag, github_release                         |
+--------------------------------------------------+
```

## Optional Config: `.unirelease.toml`

```toml
type = "rust"                    # override auto-detection
tag_prefix = "v"                 # default: "v" -> tags like v1.2.3
skip = ["clean", "test"]         # skip specific steps

[hooks]
pre_build = "make generate"      # run before build step
post_publish = "notify.sh"       # run after publish step

[commands]
build = "make release"           # override default build command
test = "make test-release"       # override default test command
```

## Version Reading

| Source | Field |
|--------|-------|
| package.json | `version` |
| Cargo.toml | `[package] version` |
| pyproject.toml | `[project] version` |

Tag format: configurable via `tag_prefix`, default `v{VERSION}`.

## MVP Scope

### In Scope
1. **Auto-detection** of Rust, Bun-binary, Node, and Python projects
2. **Version reading** from package.json, Cargo.toml, pyproject.toml
3. **Full pipeline**: detect, read_version, verify_env, check_git_status, clean, build, test, git_tag, github_release, publish
4. **CLI flags**: --step, --yes, --dry-run, --version, --type
5. **Optional .unirelease.toml** config for overrides and hooks
6. **Cross-platform**: Windows, macOS, Linux binaries via Go cross-compilation
7. **GitHub Release creation** with asset upload for binary projects

### Not in MVP
- Plugin system for custom project types
- Monorepo multi-package releases
- Changelog generation
- Semantic version bumping (version must already be set in source)
- CI/CD mode with special output formatting
- Interactive version selection

## Differentiation

| Dimension | unirelease | goreleaser | semantic-release | release-it |
|-----------|-----------|------------|-----------------|------------|
| Multi-language | **Rust + Node + Bun + Python** | Go only | Node only | Node only |
| Auto-detection | **Yes** | No | No | No |
| Single binary | **Yes (Go)** | Yes | No (npm) | No (npm) |
| Cross-platform | **Yes** | Yes | Partial | Partial |
| Zero config | **Yes** | No (.goreleaser.yml required) | No (.releaserc required) | No |
| Focus | **Release any project** | Build & release Go | Version & changelog | Version & publish |

## Success Criteria (MVP)
1. `unirelease` in a Rust project dir: detects Cargo.toml, reads version, builds, tags, publishes to crates.io
2. `unirelease` in a Bun binary project: detects bun build --compile, builds binary, creates GitHub Release with uploaded binary
3. `unirelease` in a Node project: detects package.json, builds, publishes to npm
4. `unirelease` in a Python project: detects pyproject.toml, builds, publishes to PyPI
5. `unirelease --dry-run` shows full plan without executing anything
6. Works on macOS, Linux, and Windows without modification

## Risks
1. **Edge cases in detection** -- projects with both Cargo.toml and package.json (e.g., wasm-pack projects) need clear priority rules or --type override
2. **Build tool assumptions** -- assumes pnpm for Node, may need to detect npm/yarn/bun
3. **Auth complexity** -- each registry (crates.io, npm, PyPI) has different auth mechanisms; need clear error messages when credentials are missing
4. **Scope creep** -- temptation to add changelog generation, version bumping, monorepo support

## Demand Validation Status
- [x] Problem backed by evidence (3 existing scripts, ~1,500 lines of duplicated bash)
- [x] Target users identified (immediate: the author; broader: multi-language maintainers)
- [x] Existing solutions analyzed (goreleaser, semantic-release, release-it -- none do multi-language auto-detect)
- [x] "What if we don't build this?" answered (continue maintaining 3 separate bash scripts that don't work on Windows)
- [x] Differentiation clear (only tool that auto-detects and releases Rust + Node + Bun + Python)
- [x] MVP scope defined
- [x] Name available (checked: no existing "unirelease" tool)

## Session History
- Session 1 (2026-03-21): Full idea definition -- problem, solution, CLI interface, architecture, MVP scope, validation. Status set to Ready.
