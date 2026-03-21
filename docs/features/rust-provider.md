# UNI-004: Rust Release Implementation

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-004                                            |
| **Phase**        | 2 - Language Implementations                       |
| **Priority**     | P0                                                 |
| **Effort**       | S (1-2 days)                                       |
| **Dependencies** | UNI-002, UNI-003                                   |
| **Packages**     | `internal/providers/`                              |

---

## 1. Purpose

Implement the `Provider` interface for Rust projects. This is the first language provider and proves the full pipeline end-to-end, from detection through `cargo publish` to crates.io. The implementation is modeled on the existing `rust/release.sh` (393 lines).

---

## 2. Files to Create

```
internal/
  providers/
    provider.go      (shared: Provider interface + registry)
    rust.go
    rust_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/providers/provider.go (shared)

```go
package providers

import (
    "errors"
    "fmt"
    "unirelease/internal/pipeline"
)

// ErrNoPublish indicates the provider does not publish to a registry.
// The pipeline engine treats this as a skip, not an error.
var ErrNoPublish = errors.New("provider does not publish to a registry")

// Provider defines the contract for language-specific release operations.
type Provider interface {
    Name() string
    Detect(projectDir string) (bool, int)
    ReadVersion(projectDir string) (string, error)
    VerifyEnv() ([]string, error)
    Clean(ctx *pipeline.PipelineContext) error
    Build(ctx *pipeline.PipelineContext) error
    Test(ctx *pipeline.PipelineContext) error
    Publish(ctx *pipeline.PipelineContext) error
    PublishTarget() string
    BinaryAssets(ctx *pipeline.PipelineContext) ([]string, error)
}

// ForType returns the provider for a given project type string.
func ForType(projectType string) (Provider, error) {
    switch projectType {
    case "rust":
        return &RustProvider{}, nil
    case "node":
        return &NodeProvider{}, nil
    case "bun":
        return &BunProvider{}, nil
    case "python":
        return &PythonProvider{}, nil
    default:
        return nil, fmt.Errorf("unsupported project type: %s", projectType)
    }
}
```

### 3.2 internal/providers/rust.go

```go
package providers

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/BurntSushi/toml"
    "unirelease/internal/pipeline"
)

type RustProvider struct{}

func (p *RustProvider) Name() string { return "rust" }
```

**Detect:**

```go
// Detect checks for Cargo.toml in the project directory.
// Returns (true, 100) if found, (false, 0) otherwise.
func (p *RustProvider) Detect(projectDir string) (bool, int) {
    path := filepath.Join(projectDir, "Cargo.toml")
    _, err := os.Stat(path)
    if err == nil {
        return true, 100
    }
    return false, 0
}
```

**ReadVersion:**

```go
// ReadVersion parses Cargo.toml and extracts [package].version.
func (p *RustProvider) ReadVersion(projectDir string) (string, error) {
    path := filepath.Join(projectDir, "Cargo.toml")
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read Cargo.toml: %w", err)
    }

    var cargo struct {
        Package struct {
            Version string `toml:"version"`
            Name    string `toml:"name"`
        } `toml:"package"`
    }
    if err := toml.Unmarshal(data, &cargo); err != nil {
        return "", fmt.Errorf("parse Cargo.toml: %w", err)
    }
    if cargo.Package.Version == "" {
        return "", fmt.Errorf("no version field in Cargo.toml [package] section")
    }
    return cargo.Package.Version, nil
}
```

**VerifyEnv:**

```go
// VerifyEnv checks that cargo and rustc are in PATH.
func (p *RustProvider) VerifyEnv() ([]string, error) {
    var missing []string
    for _, tool := range []string{"cargo", "rustc"} {
        if _, err := exec.LookPath(tool); err != nil {
            missing = append(missing, tool)
        }
    }
    if len(missing) > 0 {
        return missing, fmt.Errorf("missing tools: %v; install Rust from https://rustup.rs", missing)
    }
    return nil, nil
}
```

**Clean:**

```go
// Clean runs `cargo clean` in the project directory.
func (p *RustProvider) Clean(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("cargo", "clean")
    return err
}
```

**Build:**

```go
// Build runs `cargo build --release` in the project directory.
func (p *RustProvider) Build(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("cargo", "build", "--release")
    return err
}
```

**Test:**

```go
// Test runs `cargo test` in the project directory.
func (p *RustProvider) Test(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("cargo", "test")
    return err
}
```

**Publish:**

```go
// Publish runs `cargo publish` to publish the crate to crates.io.
// Requires a crates.io API token configured via `cargo login`.
func (p *RustProvider) Publish(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("cargo", "publish")
    return err
}
```

**PublishTarget and BinaryAssets:**

```go
func (p *RustProvider) PublishTarget() string { return "crates.io" }

// BinaryAssets returns nil -- Rust projects publish to a registry, not GitHub Release.
func (p *RustProvider) BinaryAssets(ctx *pipeline.PipelineContext) ([]string, error) {
    return nil, nil
}
```

---

## 4. Test Cases

### 4.1 Provider Tests (rust_test.go)

Tests use a mock `Runner` that records commands without executing them.

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestRustDetect_CargoTomlExists         | Temp dir with Cargo.toml                      | (true, 100)                         |
| TestRustDetect_NoCargoToml             | Empty temp dir                                | (false, 0)                          |
| TestRustReadVersion_Valid              | Cargo.toml: `[package]\nversion = "0.3.0"`   | "0.3.0", nil                       |
| TestRustReadVersion_MissingVersion     | Cargo.toml: `[package]\nname = "foo"`         | Error                               |
| TestRustReadVersion_MalformedToml      | Cargo.toml: invalid content                   | Parse error                          |
| TestRustReadVersion_NoFile             | Empty dir                                     | File not found error                 |
| TestRustClean_Command                  | Mock runner                                   | Ran: "cargo clean"                  |
| TestRustBuild_Command                  | Mock runner                                   | Ran: "cargo build --release"        |
| TestRustTest_Command                   | Mock runner                                   | Ran: "cargo test"                   |
| TestRustPublish_Command                | Mock runner                                   | Ran: "cargo publish"                |
| TestRustBinaryAssets_ReturnsNil        | Any                                           | nil, nil                            |
| TestRustPublishTarget                  | Any                                           | "crates.io"                         |

---

## 5. Reference: Existing rust/release.sh Mapping

| release.sh Step              | unirelease Equivalent                          |
|------------------------------|------------------------------------------------|
| Version verification (grep Cargo.toml) | `ReadVersion()` parsing TOML properly   |
| Check status (check_tag_exists, check_crates_uploaded) | `git_tag` step + verify_env  |
| Run tests (cargo test)       | `Test()` -> `cargo test`                       |
| Build release (cargo build --release) | `Build()` -> `cargo build --release`  |
| Git tag (git tag -a, git push origin) | `git_tag` step (shared, UNI-003)       |
| GitHub Release (gh release create) | `github_release` step (shared, UNI-003)   |
| Publish to crates.io (cargo publish) | `Publish()` -> `cargo publish`          |
| Summary                      | Pipeline engine prints summary                  |

---

## 6. Acceptance Criteria

- [ ] `unirelease` in a directory with Cargo.toml detects "rust".
- [ ] Version is read from Cargo.toml `[package] version`.
- [ ] `verify_env` checks for `cargo` and `rustc`; prints install URL if missing.
- [ ] `clean` runs `cargo clean`.
- [ ] `build` runs `cargo build --release`.
- [ ] `test` runs `cargo test`.
- [ ] `publish` runs `cargo publish`.
- [ ] `BinaryAssets` returns nil (no asset upload for Rust).
- [ ] Full pipeline from detection to crates.io publish works end-to-end.
- [ ] `--dry-run` shows all cargo commands without executing them.
