# Implementation Plan: UNI-001 — CLI Shell + Project Detector

| Field          | Value                                          |
|----------------|------------------------------------------------|
| **Feature**    | UNI-001                                        |
| **Phase**      | 1 - Core Pipeline                              |
| **Language**   | Go                                             |
| **Spec**       | [cli-detector.md](../features/cli-detector.md) |
| **Status**     | In Progress                                    |

---

## Tasks (TDD Order)

### Task 1: Initialize Go module and project skeleton
- `go mod init github.com/aipartnerup/unirelease`
- Create directory structure: `cmd/`, `internal/detector/`
- Add cobra dependency: `go get github.com/spf13/cobra`
- Add toml dependency: `go get github.com/BurntSushi/toml`
- Smoke test: verify module compiles

### Task 2: Implement project type detector (RED → GREEN)
- **RED**: Write `internal/detector/detector_test.go` with 9 test cases
- **GREEN**: Implement `internal/detector/detector.go`
  - `ProjectType` enum (rust, node, bun, python)
  - `DetectionResult` struct
  - `Detect(projectDir, typeOverride)` function
  - `isBunBinary(packageJSONPath)` helper
  - `ValidTypes()` function

### Task 3: Implement version reader (RED → GREEN)
- **RED**: Write `internal/detector/version_test.go` with 8 test cases
- **GREEN**: Implement `internal/detector/version.go`
  - `ReadVersion(projectDir, projectType, versionOverride)` function
  - `readCargoVersion()`, `readPackageJSONVersion()`, `readPyprojectVersion()`

### Task 4: Implement Cobra CLI with flag validation
- `cmd/root.go`: Cobra root command with all flags
- `main.go`: Entry point
- Flag validation: --step, --type, --version, path arg
- CLI tests for invalid inputs

### Task 5: Wire detection into CLI
- Connect Detect() + ReadVersion() to runRelease
- Stub PipelineContext for downstream features
- Verify all acceptance criteria
- `go test ./...` passes

---

## Acceptance Criteria

- [ ] `unirelease` in Cargo.toml dir → "Detected: rust"
- [ ] `unirelease` in package.json dir → "Detected: node"
- [ ] `unirelease` in bun-compile package.json dir → "Detected: bun"
- [ ] `unirelease` in pyproject.toml dir → "Detected: python"
- [ ] Empty dir → error exit code 3
- [ ] `--type rust` forces Rust
- [ ] `--version 1.2.3` overrides manifest version
- [ ] Cargo.toml + package.json → Cargo.toml wins
- [ ] All version readers parse correctly
