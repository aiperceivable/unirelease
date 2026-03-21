# UNI-001: CLI Shell + Project Detector

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-001                                            |
| **Phase**        | 1 - Core Pipeline                                  |
| **Priority**     | P0                                                 |
| **Effort**       | M (3-5 days)                                       |
| **Dependencies** | None                                               |
| **Packages**     | `cmd/`, `internal/detector/`                       |

---

## 1. Purpose

Provide the CLI entry point for unirelease using Cobra, parse all flags, resolve the project directory, and auto-detect the project type from manifest files. This is the first component users interact with and the foundation all other features build on.

---

## 2. Files to Create

```
main.go
cmd/
  root.go
internal/
  detector/
    detector.go
    detector_test.go
    version.go
    version_test.go
```

---

## 3. Implementation Detail

### 3.1 main.go

```go
package main

import (
    "os"
    "unirelease/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 3.2 cmd/root.go

**Cobra root command setup:**

```go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
)

var (
    flagStep    string
    flagYes     bool
    flagDryRun  bool
    flagVersion string
    flagType    string
)

var rootCmd = &cobra.Command{
    Use:   "unirelease [path]",
    Short: "Unified release pipeline for any project",
    Long:  "Auto-detects project type (Rust, Node, Bun, Python) and runs a unified release pipeline.",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runRelease,
}

func init() {
    rootCmd.Flags().StringVar(&flagStep, "step", "", "Run only a specific pipeline step")
    rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Non-interactive mode (skip confirmations)")
    rootCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview pipeline without executing")
    rootCmd.Flags().StringVarP(&flagVersion, "version", "v", "", "Override detected version")
    rootCmd.Flags().StringVar(&flagType, "type", "", "Override auto-detection (rust|node|bun|python)")
}

func Execute() error {
    return rootCmd.Execute()
}
```

**runRelease function logic:**

1. Resolve project directory:
   - If positional arg provided, use it (resolve to absolute path).
   - Otherwise, use `os.Getwd()`.
   - Validate directory exists with `os.Stat()`.
2. Load config (calls into `internal/config/` -- stubbed as empty Config in UNI-001, full implementation in UNI-007).
3. Build `PipelineContext` with all resolved values.
4. Instantiate pipeline engine, execute pipeline.
5. Map returned errors to exit codes:
   - `ErrDetection` -> exit 3
   - `ErrMissingTool` -> exit 4
   - `ErrInvalidArgs` -> exit 2
   - All other errors -> exit 1

**Flag validation in runRelease:**

| Flag        | Validation                                                                 |
|-------------|---------------------------------------------------------------------------|
| `--step`    | If non-empty, must be one of the 10 valid step names. Print valid names on error. |
| `--type`    | If non-empty, must be one of: `rust`, `node`, `bun`, `python`. Print valid types on error. |
| `--version` | If non-empty, must match semver pattern `^\d+\.\d+\.\d+` (no `v` prefix). Print format hint on error. |
| `--yes`     | Boolean, no validation needed.                                            |
| `--dry-run` | Boolean, no validation needed.                                            |
| path arg    | Must be an existing directory. Print "directory not found: <path>" on error. |

### 3.3 internal/detector/detector.go

**ProjectType enum:**

```go
package detector

type ProjectType string

const (
    TypeRust   ProjectType = "rust"
    TypeNode   ProjectType = "node"
    TypeBun    ProjectType = "bun"
    TypePython ProjectType = "python"
)

// ValidTypes returns all valid project types for flag validation.
func ValidTypes() []ProjectType {
    return []ProjectType{TypeRust, TypeNode, TypeBun, TypePython}
}
```

**DetectionResult struct:**

```go
type DetectionResult struct {
    Type       ProjectType
    Confidence int    // Higher wins when multiple detectors match
    Manifest   string // Path to the manifest file that triggered detection
}
```

**Detect function:**

```go
// Detect scans the project directory and returns the detected project type.
// If typeOverride is non-empty, it is used directly (from --type flag).
// Returns ErrNoProject if no supported manifest file is found.
func Detect(projectDir string, typeOverride string) (*DetectionResult, error)
```

**Detection logic (step by step):**

1. If `typeOverride` is non-empty:
   - Validate it is a known type.
   - Return `DetectionResult{Type: typeOverride, Confidence: 1000, Manifest: ""}`.
2. Initialize `candidates := []DetectionResult{}`.
3. Check `filepath.Join(projectDir, "Cargo.toml")` exists:
   - If yes, append `DetectionResult{Type: TypeRust, Confidence: 100, Manifest: "Cargo.toml"}`.
4. Check `filepath.Join(projectDir, "pyproject.toml")` exists:
   - If yes, append `DetectionResult{Type: TypePython, Confidence: 90, Manifest: "pyproject.toml"}`.
5. Check `filepath.Join(projectDir, "package.json")` exists:
   - If yes, read the file, parse JSON, check if any value in `.scripts` contains `"bun build --compile"`.
   - If bun build --compile found: append `DetectionResult{Type: TypeBun, Confidence: 80, Manifest: "package.json"}`.
   - Else: append `DetectionResult{Type: TypeNode, Confidence: 50, Manifest: "package.json"}`.
6. If `candidates` is empty, return `ErrNoProject`.
7. Sort candidates by `Confidence` descending.
8. Return `candidates[0]`.

**Bun detection detail:**

```go
// isBunBinary checks if any script in package.json contains "bun build --compile".
func isBunBinary(packageJSONPath string) (bool, error) {
    data, err := os.ReadFile(packageJSONPath)
    if err != nil {
        return false, err
    }
    var pkg struct {
        Scripts map[string]string `json:"scripts"`
    }
    if err := json.Unmarshal(data, &pkg); err != nil {
        return false, err
    }
    for _, script := range pkg.Scripts {
        if strings.Contains(script, "bun build --compile") {
            return true, nil
        }
    }
    return false, nil
}
```

### 3.4 internal/detector/version.go

**ReadVersion function:**

```go
// ReadVersion reads the version string from the project manifest.
// If versionOverride is non-empty, returns it directly (from --version flag).
func ReadVersion(projectDir string, projectType ProjectType, versionOverride string) (string, error)
```

**Implementation per type:**

**Rust (Cargo.toml):**
```go
func readCargoVersion(projectDir string) (string, error) {
    path := filepath.Join(projectDir, "Cargo.toml")
    var cargo struct {
        Package struct {
            Version string `toml:"version"`
        } `toml:"package"`
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read Cargo.toml: %w", err)
    }
    if err := toml.Unmarshal(data, &cargo); err != nil {
        return "", fmt.Errorf("parse Cargo.toml: %w", err)
    }
    if cargo.Package.Version == "" {
        return "", fmt.Errorf("no version found in Cargo.toml [package] section")
    }
    return cargo.Package.Version, nil
}
```

**Node/Bun (package.json):**
```go
func readPackageJSONVersion(projectDir string) (string, error) {
    path := filepath.Join(projectDir, "package.json")
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read package.json: %w", err)
    }
    var pkg struct {
        Version string `json:"version"`
    }
    if err := json.Unmarshal(data, &pkg); err != nil {
        return "", fmt.Errorf("parse package.json: %w", err)
    }
    if pkg.Version == "" {
        return "", fmt.Errorf("no 'version' field in package.json")
    }
    return pkg.Version, nil
}
```

**Python (pyproject.toml):**
```go
func readPyprojectVersion(projectDir string) (string, error) {
    path := filepath.Join(projectDir, "pyproject.toml")
    var pyproject struct {
        Project struct {
            Version string `toml:"version"`
        } `toml:"project"`
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read pyproject.toml: %w", err)
    }
    if err := toml.Unmarshal(data, &pyproject); err != nil {
        return "", fmt.Errorf("parse pyproject.toml: %w", err)
    }
    if pyproject.Project.Version == "" {
        return "", fmt.Errorf("no version found in pyproject.toml [project] section")
    }
    return pyproject.Project.Version, nil
}
```

---

## 4. Test Cases

### 4.1 Detection Tests (detector_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestDetect_Rust                        | Temp dir with Cargo.toml                      | TypeRust, Confidence 100            |
| TestDetect_Python                      | Temp dir with pyproject.toml                  | TypePython, Confidence 90           |
| TestDetect_Node                        | Temp dir with package.json (no bun compile)   | TypeNode, Confidence 50             |
| TestDetect_BunBinary                   | Temp dir with package.json containing `bun build --compile` in scripts | TypeBun, Confidence 80 |
| TestDetect_PriorityCargoOverNode       | Temp dir with both Cargo.toml and package.json | TypeRust (higher confidence)       |
| TestDetect_PriorityPythonOverNode      | Temp dir with both pyproject.toml and package.json | TypePython (higher confidence) |
| TestDetect_TypeOverride                | Temp dir with Cargo.toml, typeOverride="python" | TypePython (override wins)       |
| TestDetect_NoManifest                  | Empty temp dir                                | ErrNoProject                        |
| TestDetect_InvalidTypeOverride         | typeOverride="java"                           | Error: unsupported type             |

### 4.2 Version Tests (version_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestReadVersion_Cargo                  | Cargo.toml with `version = "0.3.0"`          | "0.3.0"                             |
| TestReadVersion_PackageJSON            | package.json with `"version": "1.2.0"`       | "1.2.0"                             |
| TestReadVersion_Pyproject              | pyproject.toml with `version = "2.0.0"`      | "2.0.0"                             |
| TestReadVersion_Override               | Any manifest, versionOverride="9.9.9"        | "9.9.9"                             |
| TestReadVersion_MissingVersion_Cargo   | Cargo.toml without version field             | Error                                |
| TestReadVersion_MissingVersion_JSON    | package.json without version field           | Error                                |
| TestReadVersion_MalformedTOML          | Invalid TOML content                         | Parse error                          |
| TestReadVersion_MalformedJSON          | Invalid JSON content                         | Parse error                          |

### 4.3 CLI Tests

| Test Name                              | Args                                          | Expected Behavior                   |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestCLI_NoArgs                         | `unirelease`                                  | Uses cwd, runs detection            |
| TestCLI_ExplicitPath                   | `unirelease /tmp/project`                     | Uses /tmp/project                   |
| TestCLI_InvalidPath                    | `unirelease /nonexistent`                     | Exit code 2, "not found" message    |
| TestCLI_InvalidStep                    | `unirelease --step bogus`                     | Exit code 2, lists valid steps      |
| TestCLI_InvalidType                    | `unirelease --type java`                      | Exit code 2, lists valid types      |
| TestCLI_InvalidVersion                 | `unirelease --version abc`                    | Exit code 2, format hint            |
| TestCLI_DryRunFlag                     | `unirelease --dry-run`                        | Sets DryRun=true on context         |
| TestCLI_YesFlag                        | `unirelease -y`                               | Sets Yes=true on context            |

---

## 5. Acceptance Criteria

- [ ] `unirelease` in a directory with Cargo.toml prints "Detected: rust" and proceeds.
- [ ] `unirelease` in a directory with package.json (no bun compile) prints "Detected: node".
- [ ] `unirelease` in a directory with package.json containing `bun build --compile` prints "Detected: bun".
- [ ] `unirelease` in a directory with pyproject.toml prints "Detected: python".
- [ ] `unirelease` in an empty directory prints error with supported types and exits with code 3.
- [ ] `unirelease --type rust` in any directory forces Rust detection.
- [ ] `unirelease --version 1.2.3` overrides the version read from manifest.
- [ ] `unirelease /path/to/project` resolves to the specified path.
- [ ] Version is correctly read from Cargo.toml `[package] version`, package.json `version`, pyproject.toml `[project] version`.
- [ ] When both Cargo.toml and package.json exist, Cargo.toml wins.
