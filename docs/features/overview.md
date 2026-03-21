# Feature Specs: unirelease

| Field        | Value                                                              |
|--------------|--------------------------------------------------------------------|
| **Source**   | [Tech Design](../unirelease/tech-design.md)                       |
| **Date**     | 2026-03-21                                                         |
| **Features** | 8                                                                  |

---

## Feature Index

| ID      | Feature                          | Spec                                          | Phase | Priority | Effort |
|---------|----------------------------------|-----------------------------------------------|-------|----------|--------|
| UNI-001 | CLI Shell + Project Detector     | [cli-detector.md](cli-detector.md)             | 1     | P0       | M      |
| UNI-002 | Pipeline Engine                  | [pipeline-engine.md](pipeline-engine.md)       | 1     | P0       | M      |
| UNI-003 | Git + GitHub Operations          | [git-github.md](git-github.md)                 | 1     | P0       | M      |
| UNI-004 | Rust Release Implementation      | [rust-provider.md](rust-provider.md)           | 2     | P0       | S      |
| UNI-005 | Node + Bun-Binary Release Impls  | [node-bun-provider.md](node-bun-provider.md)   | 2     | P0       | M      |
| UNI-006 | Python Release Implementation    | [python-provider.md](python-provider.md)       | 2     | P0       | S      |
| UNI-007 | Config File Support              | [config-file.md](config-file.md)               | 3     | P1       | S      |
| UNI-008 | Interactive Prompts + UX Polish  | [interactive-ux.md](interactive-ux.md)         | 3     | P1       | S      |

---

## Dependency Graph

```
UNI-001 (CLI + Detector)
  └── UNI-002 (Pipeline Engine)
        ├── UNI-003 (Git + GitHub)
        │     ├── UNI-004 (Rust Provider)
        │     ├── UNI-005 (Node + Bun Providers)
        │     └── UNI-006 (Python Provider)
        ├── UNI-007 (Config File)
        └── UNI-008 (Interactive Prompts)
```

---

## Critical Path

```
UNI-001 -> UNI-002 -> UNI-003 -> UNI-004 (proves full pipeline end-to-end with first language)
```

---

## Execution Order

| Sprint | Features                          | Deliverable                                      |
|--------|-----------------------------------|--------------------------------------------------|
| 1      | UNI-001, UNI-002                  | `unirelease --dry-run` detects type, prints plan |
| 2      | UNI-003, UNI-004                  | Full Rust release end-to-end                     |
| 3      | UNI-005, UNI-006                  | All 4 language types supported                   |
| 4      | UNI-007, UNI-008                  | Config overrides + interactive prompts           |

---

## Shared Infrastructure

| Component            | Package                | Built In   | Used By                    |
|----------------------|------------------------|------------|----------------------------|
| CLI flag parsing     | `cmd/root.go`          | UNI-001    | All                        |
| Project detector     | `internal/detector/`   | UNI-001    | UNI-001, UNI-002           |
| Version reader       | `internal/detector/`   | UNI-001    | UNI-001, all providers     |
| Pipeline orchestrator| `internal/pipeline/`   | UNI-002    | All                        |
| Git operations       | `internal/git/`        | UNI-003    | UNI-003, all providers     |
| GitHub client        | `internal/github/`     | UNI-003    | UNI-003, UNI-005           |
| Command runner       | `internal/runner/`     | UNI-002    | All providers, UNI-007     |
| Config loader        | `internal/config/`     | UNI-007    | UNI-007, UNI-002           |
| UI (colors, prompts) | `internal/ui/`         | UNI-008    | All                        |
