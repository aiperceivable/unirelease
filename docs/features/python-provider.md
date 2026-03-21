# UNI-006: Python Release Implementation

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-006                                            |
| **Phase**        | 2 - Language Implementations                       |
| **Priority**     | P0                                                 |
| **Effort**       | S (1-2 days)                                       |
| **Dependencies** | UNI-002, UNI-003                                   |
| **Packages**     | `internal/providers/`                              |

---

## 1. Purpose

Implement the `Provider` interface for Python projects using pyproject.toml. Handles version reading from `[project] version`, building with `python -m build`, testing with `pytest`, and publishing with `twine upload`. Modeled on the existing `python/src/apdev/release.sh` (~500 lines).

---

## 2. Files to Create

```
internal/
  providers/
    python.go
    python_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/providers/python.go

```go
package providers

type PythonProvider struct{}

func (p *PythonProvider) Name() string { return "python" }
```

**Detect:**

```go
// Detect checks for pyproject.toml in the project directory.
// Returns (true, 90) if found.
func (p *PythonProvider) Detect(projectDir string) (bool, int) {
    path := filepath.Join(projectDir, "pyproject.toml")
    _, err := os.Stat(path)
    if err == nil {
        return true, 90
    }
    return false, 0
}
```

**ReadVersion:**

```go
// ReadVersion parses pyproject.toml and extracts [project].version.
// Handles the standard pyproject.toml format:
//   [project]
//   version = "1.2.3"
func (p *PythonProvider) ReadVersion(projectDir string) (string, error) {
    path := filepath.Join(projectDir, "pyproject.toml")
    data, err := os.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("read pyproject.toml: %w", err)
    }

    var pyproject struct {
        Project struct {
            Version string `toml:"version"`
            Name    string `toml:"name"`
        } `toml:"project"`
    }
    if err := toml.Unmarshal(data, &pyproject); err != nil {
        return "", fmt.Errorf("parse pyproject.toml: %w", err)
    }
    if pyproject.Project.Version == "" {
        return "", fmt.Errorf("no version field in pyproject.toml [project] section; is version dynamic?")
    }
    return pyproject.Project.Version, nil
}
```

**VerifyEnv:**

```go
// VerifyEnv checks that python (or python3), and twine are available.
// Also checks that the `build` module is importable.
func (p *PythonProvider) VerifyEnv() ([]string, error) {
    var missing []string

    // Check python or python3
    pythonCmd := resolvePythonCommand()
    if pythonCmd == "" {
        missing = append(missing, "python/python3")
    }

    // Check twine (needed for PyPI upload)
    if _, err := exec.LookPath("twine"); err != nil {
        missing = append(missing, "twine (pip install twine)")
    }

    // Check python -m build is available
    if pythonCmd != "" {
        cmd := exec.Command(pythonCmd, "-m", "build", "--version")
        if err := cmd.Run(); err != nil {
            missing = append(missing, "build (pip install build)")
        }
    }

    if len(missing) > 0 {
        return missing, fmt.Errorf("missing tools: %v", missing)
    }
    return nil, nil
}

// resolvePythonCommand returns "python3" or "python", whichever is available.
// Returns "" if neither is found.
func resolvePythonCommand() string {
    if _, err := exec.LookPath("python3"); err == nil {
        return "python3"
    }
    if _, err := exec.LookPath("python"); err == nil {
        return "python"
    }
    return ""
}
```

**Clean:**

```go
// Clean removes dist/, build/, and *.egg-info/ directories.
// Uses os.RemoveAll and filepath.Glob for cross-platform compatibility.
func (p *PythonProvider) Clean(ctx *pipeline.PipelineContext) error {
    // Fixed directories
    for _, dir := range []string{"dist", "build", ".eggs"} {
        path := filepath.Join(ctx.ProjectDir, dir)
        if err := os.RemoveAll(path); err != nil {
            return fmt.Errorf("clean %s: %w", dir, err)
        }
    }

    // Glob for *.egg-info directories
    matches, _ := filepath.Glob(filepath.Join(ctx.ProjectDir, "*.egg-info"))
    for _, match := range matches {
        if err := os.RemoveAll(match); err != nil {
            return fmt.Errorf("clean %s: %w", match, err)
        }
    }

    // Also check src/ layout: src/*.egg-info
    srcMatches, _ := filepath.Glob(filepath.Join(ctx.ProjectDir, "src", "*.egg-info"))
    for _, match := range srcMatches {
        if err := os.RemoveAll(match); err != nil {
            return fmt.Errorf("clean %s: %w", match, err)
        }
    }

    ctx.UI.Info("Cleaned dist/, build/, *.egg-info/")
    return nil
}
```

**Build:**

```go
// Build runs `python -m build` to create sdist and wheel distributions.
func (p *PythonProvider) Build(ctx *pipeline.PipelineContext) error {
    pythonCmd := resolvePythonCommand()
    if pythonCmd == "" {
        return fmt.Errorf("python not found")
    }
    _, err := ctx.Runner.Run(pythonCmd, "-m", "build")
    return err
}
```

**Test:**

```go
// Test runs pytest. Checks for pytest in PATH first, falls back to python -m pytest.
func (p *PythonProvider) Test(ctx *pipeline.PipelineContext) error {
    if _, err := exec.LookPath("pytest"); err == nil {
        _, err := ctx.Runner.Run("pytest")
        return err
    }
    pythonCmd := resolvePythonCommand()
    if pythonCmd == "" {
        return fmt.Errorf("python not found")
    }
    _, err := ctx.Runner.Run(pythonCmd, "-m", "pytest")
    return err
}
```

**Publish:**

```go
// Publish runs `twine upload dist/*` to upload to PyPI.
// Twine uses credentials from ~/.pypirc, TWINE_USERNAME/TWINE_PASSWORD env vars,
// or keyring.
func (p *PythonProvider) Publish(ctx *pipeline.PipelineContext) error {
    // Glob for dist files
    distDir := filepath.Join(ctx.ProjectDir, "dist")
    patterns := []string{
        filepath.Join(distDir, "*.whl"),
        filepath.Join(distDir, "*.tar.gz"),
    }
    var files []string
    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        files = append(files, matches...)
    }
    if len(files) == 0 {
        return fmt.Errorf("no distribution files in dist/; run build first")
    }

    args := append([]string{"upload"}, files...)
    _, err := ctx.Runner.Run("twine", args...)
    return err
}

func (p *PythonProvider) PublishTarget() string { return "PyPI" }

func (p *PythonProvider) BinaryAssets(ctx *pipeline.PipelineContext) ([]string, error) {
    return nil, nil
}
```

---

## 4. Test Cases

### 4.1 Provider Tests (python_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestPythonDetect_PyprojectExists       | Temp dir with pyproject.toml                  | (true, 90)                          |
| TestPythonDetect_NoPyproject           | Empty temp dir                                | (false, 0)                          |
| TestPythonReadVersion_Valid            | pyproject.toml: `[project]\nversion = "2.0.0"` | "2.0.0"                          |
| TestPythonReadVersion_DynamicVersion   | pyproject.toml with `dynamic = ["version"]`   | Error: "is version dynamic?"        |
| TestPythonReadVersion_MissingVersion   | pyproject.toml: `[project]\nname = "foo"`     | Error                               |
| TestPythonReadVersion_MalformedToml    | Invalid TOML                                  | Parse error                          |
| TestPythonClean_RemovesDirs            | Dir with dist/, build/, foo.egg-info/         | All removed                         |
| TestPythonClean_SrcLayout              | Dir with src/foo.egg-info/                    | Removed                             |
| TestPythonBuild_Command                | Mock runner                                   | Ran: "python3 -m build" (or "python") |
| TestPythonTest_PytestInPath            | pytest available                              | Ran: "pytest"                       |
| TestPythonTest_FallbackPythonM         | pytest not in PATH                            | Ran: "python3 -m pytest"           |
| TestPythonPublish_Command              | Mock runner, dist/ has .whl and .tar.gz       | Ran: "twine upload <files>"         |
| TestPythonPublish_NoDistFiles          | Empty dist/                                   | Error: "no distribution files"      |
| TestPythonBinaryAssets_ReturnsNil      | Any                                           | nil                                 |
| TestResolvePythonCommand_Python3       | python3 in PATH                               | "python3"                           |
| TestResolvePythonCommand_Python        | Only python in PATH                           | "python"                            |

---

## 5. Reference: Existing python/release.sh Mapping

| release.sh Step                        | unirelease Python Provider Equivalent           |
|----------------------------------------|--------------------------------------------------|
| Auto-detect name from pyproject.toml   | `Detect()` + `ReadVersion()`                     |
| Determine package name (dash to underscore) | Not needed (only used for import check)     |
| Version verification (pyproject.toml + __init__.py) | `ReadVersion()` reads pyproject.toml only; __init__.py check omitted (low value, complex) |
| Check status (check_tag_exists, check_pypi_uploaded) | git_tag step + verify_env                |
| Clean (rm -rf dist/ build/ *.egg-info/) | `Clean()` -> os.RemoveAll + Glob               |
| Build (python -m build)               | `Build()` -> `python -m build`                   |
| Check (twine check dist/*)            | Omitted in MVP (twine check is informational)    |
| Git tag                                | Shared git_tag step (UNI-003)                    |
| GitHub Release (gh/API)               | Shared github_release step (UNI-003)             |
| Upload to PyPI (twine upload)          | `Publish()` -> `twine upload dist/*.whl dist/*.tar.gz` |

**Simplifications from original script:**
- No `__init__.py` version cross-check (complex for dynamic versioning, low value for release flow).
- No `twine check` step (informational only; twine upload will fail if package is broken).
- No dual API/gh-CLI fallback for GitHub Releases (handled centrally by UNI-003).

---

## 6. Acceptance Criteria

- [ ] `unirelease` with pyproject.toml detects "python".
- [ ] Version is read from pyproject.toml `[project] version`.
- [ ] Clear error when version is dynamic (not set in pyproject.toml).
- [ ] `verify_env` checks for python/python3, twine, and build module.
- [ ] `clean` removes dist/, build/, *.egg-info/ (including src/ layout).
- [ ] `build` runs `python -m build` (using python3 if available, python otherwise).
- [ ] `test` runs `pytest` if available, falls back to `python -m pytest`.
- [ ] `publish` runs `twine upload` with the actual dist/ files (not a wildcard string).
- [ ] `publish` errors if dist/ is empty (no files to upload).
- [ ] `BinaryAssets` returns nil.
- [ ] Full pipeline from detection to PyPI upload works end-to-end.
