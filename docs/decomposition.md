# Project Decomposition: unirelease

| Field        | Value                                      |
|--------------|--------------------------------------------|
| **Source**   | [Idea Draft](../ideas/unirelease/draft.md) |
| **Date**     | 2026-03-21                                 |
| **Type**     | Multi-feature project                      |
| **Features** | 8 sub-features across 3 phases             |

---

## Decomposition Verdict

**Multi-split project.** unirelease has 8 distinct features organized into 3 phases. The system decomposes cleanly along architectural boundaries: a CLI shell, a detection/version layer, a pipeline engine, language-specific implementations, and cross-cutting concerns (config, dry-run, interactive prompts). The pipeline engine is the central coordination point -- it depends on detection and is consumed by all language implementations. Config and Git/GitHub operations are shared infrastructure used across the pipeline.

**Rationale for split (not single feature)**:
- The draft identifies 5 distinct architectural layers (CLI, Detector, Pipeline Engine, Language Impls, Shared Steps)
- Language-specific implementations (Rust, Node, Bun-binary, Python) are independently testable and shippable
- Config file support, dry-run mode, and interactive prompts are orthogonal concerns that layer on top of core functionality
- A single feature would exceed 2 weeks of effort and lack meaningful intermediate checkpoints

---

## Feature Manifest

### Phase 1: Core Pipeline (v0.1 -- Minimum Viable Release)

| ID       | Feature                              | Priority | Dependencies | Effort | Description |
|----------|--------------------------------------|----------|--------------|--------|-------------|
| UNI-001  | CLI Shell + Project Detector         | P0       | None         | M      | Cobra CLI with all flags (--step, --yes, --dry-run, --version, --type), project path resolution, file-based auto-detection (Cargo.toml, package.json, pyproject.toml), version reading from each manifest format |
| UNI-002  | Pipeline Engine                      | P0       | UNI-001      | M      | Step orchestration (detect, read_version, verify_env, check_git_status, clean, build, test, git_tag, github_release, publish), --step filtering, --dry-run preview, step sequencing and error handling |
| UNI-003  | Git + GitHub Operations              | P0       | UNI-002      | M      | Shared steps: verify clean working tree, verify branch, create and push git tag, create GitHub Release via API, upload binary assets for bun-binary projects |

### Phase 2: Language Implementations (v0.2 -- Full Coverage)

| ID       | Feature                              | Priority | Dependencies       | Effort | Description |
|----------|--------------------------------------|----------|--------------------|--------|-------------|
| UNI-004  | Rust Release Implementation          | P0       | UNI-002, UNI-003   | S      | verify_env (cargo, rustc), clean (cargo clean), build (cargo build --release), test (cargo test), publish (cargo publish to crates.io) |
| UNI-005  | Node + Bun-Binary Release Impls      | P0       | UNI-002, UNI-003   | M      | Node: verify_env (pnpm/npm), clean (rm dist/), build (pnpm build), test (pnpm test), publish (npm publish). Bun-binary: detect `bun build --compile` in scripts, build binary, upload to GitHub Release instead of registry publish |
| UNI-006  | Python Release Implementation        | P0       | UNI-002, UNI-003   | S      | verify_env (python, pip/uv, twine), clean (rm dist/), build (python -m build), test (pytest), publish (twine upload to PyPI) |

### Phase 3: Polish (v0.3 -- Production Ready)

| ID       | Feature                              | Priority | Dependencies       | Effort | Description |
|----------|--------------------------------------|----------|--------------------|--------|-------------|
| UNI-007  | Config File Support (.unirelease.toml) | P1     | UNI-002            | S      | TOML parsing, type/tag_prefix overrides, skip steps, pre/post hooks, command overrides for build/test |
| UNI-008  | Interactive Prompts + UX Polish      | P1       | UNI-002            | S      | Confirmation prompts before destructive steps (tag, publish), --yes to skip, colored output, step progress display, summary report at end |

---

## Dependency Graph

```
UNI-001 (CLI + Detector) ─────────────────────────────────────┐
  └── UNI-002 (Pipeline Engine) ──────────────────────────────┤
        ├── UNI-003 (Git + GitHub) ───────────────────────────┤
        │     ├── UNI-004 (Rust Impl)                         │
        │     ├── UNI-005 (Node + Bun Impls)                  │
        │     └── UNI-006 (Python Impl)                       │
        ├── UNI-007 (Config File)                              │
        └── UNI-008 (Interactive Prompts)                      │
```

---

## Critical Path (MVP)

```
UNI-001 → UNI-002 → UNI-003 → UNI-004 (first language proves the pipeline end-to-end)
```

**Rationale**: Rust is the simplest language implementation (single build tool, single registry) and is sufficient to validate the entire pipeline from detection through publishing. Once UNI-004 works, UNI-005 and UNI-006 are additive.

---

## Execution Order

| Sprint | Tasks | Deliverable |
|--------|-------|-------------|
| 1 | **UNI-001** CLI Shell + Detector, **UNI-002** Pipeline Engine | `unirelease --dry-run` detects project type and prints planned steps |
| 2 | **UNI-003** Git + GitHub, **UNI-004** Rust Release | Full release of a Rust project end-to-end |
| 3 | **UNI-005** Node + Bun, **UNI-006** Python | All 4 language types supported |
| 4 | **UNI-007** Config File, **UNI-008** Interactive Prompts | Production-ready UX with overrides and confirmations |

**Estimated total: 3-4 weeks**

---

## Shared Infrastructure (built in UNI-001/002/003, reused by all)

| Component | Package | Used By |
|-----------|---------|---------|
| CLI flag parsing | `cmd/root.go` (cobra) | All features |
| Project detector | `internal/detect/` | UNI-001, UNI-002 |
| Version reader | `internal/version/` | UNI-001, all language impls |
| Pipeline orchestrator | `internal/pipeline/` | All features |
| Git operations | `internal/git/` | UNI-003, all language impls |
| GitHub Release client | `internal/github/` | UNI-003, UNI-005 (binary upload) |
| Command runner (exec) | `internal/runner/` | All language impls, UNI-007 hooks |
| Config loader | `internal/config/` | UNI-007, UNI-002 |

---

## Design Decisions

1. **Node and Bun-binary are combined in UNI-005** because they share the same detection file (package.json) and diverge only on the presence of `bun build --compile` in scripts. The detection logic and most steps overlap.
2. **Config file (UNI-007) is Phase 3, not Phase 1** because the draft specifies it as optional and the CLI flags already cover overrides. Zero-config is a core differentiator.
3. **Interactive prompts (UNI-008) are Phase 3** because --yes mode is the safer default for initial development and testing. Prompts are UX polish, not core functionality.
4. **Git + GitHub is a separate feature (UNI-003)** rather than embedded in each language impl because git_tag, check_git_status, and github_release are shared steps identical across all languages.

---

## Effort Legend

| Size | Estimate    | Description                             |
|------|-------------|-----------------------------------------|
| S    | 1-2 days    | Single module, limited scope            |
| M    | 3-5 days    | Multiple modules, integration points    |

---

## Open Questions

1. **Build tool detection for Node**: The draft assumes pnpm but notes npm/yarn/bun as a risk. UNI-005 should detect the package manager from lockfile presence (pnpm-lock.yaml, package-lock.json, yarn.lock, bun.lockb).
2. **Detection priority**: When multiple manifest files exist (e.g., Cargo.toml + package.json in wasm-pack projects), what is the priority order? Recommendation: Cargo.toml > pyproject.toml > package.json, overridable via --type.
3. **GitHub Release auth**: Use `GITHUB_TOKEN` env var or `gh` CLI auth? Recommendation: check for `GITHUB_TOKEN` first, fall back to `gh auth status`.

---

*Next: spec-forge:tech-design for UNI-001 + UNI-002, then code-forge:plan for implementation.*
