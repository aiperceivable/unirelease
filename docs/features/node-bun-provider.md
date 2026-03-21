# UNI-005: Node + Bun-Binary Release Implementations

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-005                                            |
| **Phase**        | 2 - Language Implementations                       |
| **Priority**     | P0                                                 |
| **Effort**       | M (3-5 days)                                       |
| **Dependencies** | UNI-002, UNI-003                                   |
| **Packages**     | `internal/providers/`                              |

---

## 1. Purpose

Implement the `Provider` interface for Node.js and Bun-binary projects. These two providers share the same detection file (package.json) and are combined in one feature because their detection logic, version reading, and most infrastructure overlap. They diverge on:
- Build commands (pnpm/npm/yarn vs bun).
- Publish target (npm registry vs GitHub Release asset upload).
- Package manager detection (Node only).

---

## 2. Files to Create

```
internal/
  providers/
    node.go
    node_test.go
    bun.go
    bun_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/providers/node.go

```go
package providers

type NodeProvider struct {
    packageManager string // resolved during VerifyEnv: "pnpm", "npm", "yarn", "bun"
}

func (p *NodeProvider) Name() string { return "node" }
```

**Detect:**

```go
// Detect checks for package.json without "bun build --compile" in scripts.
// Returns (true, 50) if package.json exists and is NOT a bun-binary project.
func (p *NodeProvider) Detect(projectDir string) (bool, int) {
    pkgPath := filepath.Join(projectDir, "package.json")
    if _, err := os.Stat(pkgPath); err != nil {
        return false, 0
    }
    isBun, _ := isBunBinary(pkgPath)
    if isBun {
        return false, 0 // Bun provider handles this
    }
    return true, 50
}
```

**ReadVersion:**

```go
// ReadVersion parses package.json and returns the "version" field.
func (p *NodeProvider) ReadVersion(projectDir string) (string, error) {
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

**Package Manager Detection:**

```go
// detectPackageManager determines the package manager from lockfile presence.
// Priority: pnpm-lock.yaml > bun.lockb > yarn.lock > package-lock.json > npm default.
func detectPackageManager(projectDir string) string {
    lockfiles := []struct {
        file    string
        manager string
    }{
        {"pnpm-lock.yaml", "pnpm"},
        {"bun.lockb", "bun"},
        {"yarn.lock", "yarn"},
        {"package-lock.json", "npm"},
    }
    for _, lf := range lockfiles {
        if _, err := os.Stat(filepath.Join(projectDir, lf.file)); err == nil {
            return lf.manager
        }
    }
    return "npm" // default
}
```

**VerifyEnv:**

```go
// VerifyEnv detects the package manager and checks it is installed.
// Also checks that `npm` is available (needed for npm publish regardless of build tool).
func (p *NodeProvider) VerifyEnv() ([]string, error) {
    // Detect package manager (stored for later use by Build/Test)
    p.packageManager = detectPackageManager(ctx.ProjectDir)

    var missing []string
    // Check package manager
    if _, err := exec.LookPath(p.packageManager); err != nil {
        missing = append(missing, p.packageManager)
    }
    // npm is always needed for publish
    if p.packageManager != "npm" {
        if _, err := exec.LookPath("npm"); err != nil {
            missing = append(missing, "npm")
        }
    }
    if len(missing) > 0 {
        return missing, fmt.Errorf("missing tools: %v", missing)
    }
    return nil, nil
}
```

**Clean:**

```go
// Clean removes dist/ and node_modules/.cache/ directories.
// Uses os.RemoveAll for cross-platform compatibility.
func (p *NodeProvider) Clean(ctx *pipeline.PipelineContext) error {
    dirs := []string{
        filepath.Join(ctx.ProjectDir, "dist"),
        filepath.Join(ctx.ProjectDir, "node_modules", ".cache"),
    }
    for _, dir := range dirs {
        if err := os.RemoveAll(dir); err != nil {
            return fmt.Errorf("clean %s: %w", dir, err)
        }
    }
    ctx.UI.Info("Cleaned dist/ and node_modules/.cache/")
    return nil
}
```

**Build:**

```go
// Build runs `<package-manager> build` or `<package-manager> run build`.
// pnpm/bun/yarn: `<pm> build` works.
// npm: `npm run build`.
func (p *NodeProvider) Build(ctx *pipeline.PipelineContext) error {
    pm := p.packageManager
    if pm == "" {
        pm = "npm"
    }
    args := []string{"build"}
    if pm == "npm" {
        args = []string{"run", "build"}
    }
    _, err := ctx.Runner.Run(pm, args...)
    return err
}
```

**Test:**

```go
// Test runs `<package-manager> test`.
func (p *NodeProvider) Test(ctx *pipeline.PipelineContext) error {
    pm := p.packageManager
    if pm == "" {
        pm = "npm"
    }
    args := []string{"test"}
    if pm == "npm" {
        args = []string{"run", "test"}
    }
    _, err := ctx.Runner.Run(pm, args...)
    return err
}
```

**Publish:**

```go
// Publish runs `npm publish --access public`.
// Always uses npm regardless of the build-time package manager.
func (p *NodeProvider) Publish(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("npm", "publish", "--access", "public")
    return err
}

func (p *NodeProvider) PublishTarget() string { return "npm" }

func (p *NodeProvider) BinaryAssets(ctx *pipeline.PipelineContext) ([]string, error) {
    return nil, nil
}
```

### 3.2 internal/providers/bun.go

```go
package providers

type BunProvider struct{}

func (p *BunProvider) Name() string { return "bun" }
```

**Detect:**

```go
// Detect checks for package.json with "bun build --compile" in any script value.
// Returns (true, 80) if found.
func (p *BunProvider) Detect(projectDir string) (bool, int) {
    pkgPath := filepath.Join(projectDir, "package.json")
    if _, err := os.Stat(pkgPath); err != nil {
        return false, 0
    }
    isBun, _ := isBunBinary(pkgPath)
    if isBun {
        return true, 80
    }
    return false, 0
}
```

**Shared helper (in provider.go or a helpers.go):**

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

**ReadVersion:** Same as NodeProvider (parse package.json `.version`).

**VerifyEnv:**

```go
// VerifyEnv checks that `bun` is in PATH.
func (p *BunProvider) VerifyEnv() ([]string, error) {
    if _, err := exec.LookPath("bun"); err != nil {
        return []string{"bun"}, fmt.Errorf("bun not found; install from https://bun.sh")
    }
    return nil, nil
}
```

**Clean:**

```go
// Clean removes the dist/ directory.
func (p *BunProvider) Clean(ctx *pipeline.PipelineContext) error {
    dir := filepath.Join(ctx.ProjectDir, "dist")
    return os.RemoveAll(dir)
}
```

**Build:**

```go
// Build runs `bun run build` to compile the binary.
func (p *BunProvider) Build(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("bun", "run", "build")
    return err
}
```

**Test:**

```go
// Test runs `bun test`.
func (p *BunProvider) Test(ctx *pipeline.PipelineContext) error {
    _, err := ctx.Runner.Run("bun", "test")
    return err
}
```

**Publish:**

```go
// Publish returns ErrNoPublish -- bun-binary projects upload to GitHub Release,
// not to a package registry. The asset upload is handled by the github_release step.
func (p *BunProvider) Publish(ctx *pipeline.PipelineContext) error {
    return ErrNoPublish
}

func (p *BunProvider) PublishTarget() string { return "GitHub Release" }
```

**BinaryAssets:**

```go
// BinaryAssets returns the compiled binary path(s) for GitHub Release upload.
// Strategy:
// 1. Parse the build script to find --outfile value.
// 2. If --outfile not found, scan dist/ for executable files.
func (p *BunProvider) BinaryAssets(ctx *pipeline.PipelineContext) ([]string, error) {
    // Strategy 1: Parse --outfile from build script
    pkgPath := filepath.Join(ctx.ProjectDir, "package.json")
    data, _ := os.ReadFile(pkgPath)
    var pkg struct {
        Scripts map[string]string `json:"scripts"`
    }
    json.Unmarshal(data, &pkg)

    for _, script := range pkg.Scripts {
        if strings.Contains(script, "bun build --compile") {
            // Look for --outfile <path>
            parts := strings.Fields(script)
            for i, part := range parts {
                if part == "--outfile" && i+1 < len(parts) {
                    outfile := parts[i+1]
                    absPath := filepath.Join(ctx.ProjectDir, outfile)
                    if _, err := os.Stat(absPath); err == nil {
                        return []string{absPath}, nil
                    }
                }
            }
        }
    }

    // Strategy 2: Scan dist/ for executable files
    distDir := filepath.Join(ctx.ProjectDir, "dist")
    entries, err := os.ReadDir(distDir)
    if err != nil {
        return nil, fmt.Errorf("no binary found: dist/ not found and --outfile not parsed")
    }
    var assets []string
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        info, _ := entry.Info()
        // Check if file is executable (Unix) or has no extension (could be binary)
        if info.Mode()&0111 != 0 || filepath.Ext(entry.Name()) == "" {
            assets = append(assets, filepath.Join(distDir, entry.Name()))
        }
    }
    if len(assets) == 0 {
        return nil, fmt.Errorf("no binary assets found in dist/")
    }
    return assets, nil
}
```

---

## 4. Test Cases

### 4.1 Node Provider Tests (node_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestNodeDetect_PackageJSON             | package.json without bun compile              | (true, 50)                          |
| TestNodeDetect_BunBinaryPackageJSON    | package.json with bun build --compile         | (false, 0)                          |
| TestNodeDetect_NoPackageJSON           | Empty dir                                     | (false, 0)                          |
| TestNodeReadVersion_Valid              | package.json: `{"version": "1.2.0"}`         | "1.2.0"                             |
| TestNodeReadVersion_NoVersion          | package.json: `{"name": "foo"}`              | Error                                |
| TestDetectPM_Pnpm                      | Dir with pnpm-lock.yaml                       | "pnpm"                              |
| TestDetectPM_Bun                       | Dir with bun.lockb                            | "bun"                               |
| TestDetectPM_Yarn                      | Dir with yarn.lock                            | "yarn"                              |
| TestDetectPM_Npm                       | Dir with package-lock.json                    | "npm"                               |
| TestDetectPM_Default                   | Dir with no lockfile                          | "npm"                               |
| TestDetectPM_Priority                  | Dir with pnpm-lock.yaml AND package-lock.json | "pnpm"                             |
| TestNodeClean_RemovesDirs              | Dir with dist/ and node_modules/.cache/       | Both removed                        |
| TestNodeBuild_Pnpm                     | packageManager = "pnpm"                       | Ran: "pnpm build"                   |
| TestNodeBuild_Npm                      | packageManager = "npm"                        | Ran: "npm run build"                |
| TestNodePublish_Command                | Mock runner                                   | Ran: "npm publish --access public"  |
| TestNodeBinaryAssets_ReturnsNil        | Any                                           | nil                                 |

### 4.2 Bun Provider Tests (bun_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestBunDetect_WithCompile              | package.json: scripts.build = "bun build --compile ..." | (true, 80)            |
| TestBunDetect_WithoutCompile           | package.json: scripts.build = "bun build ..." (no --compile) | (false, 0)        |
| TestBunDetect_NoPackageJSON            | Empty dir                                     | (false, 0)                          |
| TestBunBuild_Command                   | Mock runner                                   | Ran: "bun run build"               |
| TestBunTest_Command                    | Mock runner                                   | Ran: "bun test"                     |
| TestBunPublish_ReturnsErrNoPublish     | Any                                           | ErrNoPublish                        |
| TestBunBinaryAssets_OutfileFlag        | package.json: `--outfile dist/myapp`, file exists | ["<abs>/dist/myapp"]          |
| TestBunBinaryAssets_ScanDist           | dist/ contains executable file                | Returns that file path              |
| TestBunBinaryAssets_NoBinary           | Empty dist/ or no dist/                       | Error                                |

---

## 5. Reference: Existing typescript/release.sh Mapping

| release.sh Step                 | unirelease Node Provider Equivalent              |
|---------------------------------|---------------------------------------------------|
| Auto-detect name from package.json | `ReadVersion()` + pipeline detect step          |
| Install deps (pnpm install)     | Not in MVP (assume deps installed)                |
| Clean (rm -rf dist/)            | `Clean()` -> os.RemoveAll                         |
| Build (pnpm build)              | `Build()` -> `<pm> build`                         |
| Check (npm pack --dry-run)      | Omitted in MVP (low value)                        |
| Git tag                         | Shared git_tag step (UNI-003)                     |
| GitHub Release                  | Shared github_release step (UNI-003)              |
| npm publish                     | `Publish()` -> `npm publish --access public`      |

---

## 6. Acceptance Criteria

- [ ] Node: `unirelease` with package.json (no bun compile) detects "node".
- [ ] Bun: `unirelease` with package.json containing `bun build --compile` detects "bun".
- [ ] Node: Package manager detected from lockfile (pnpm-lock.yaml -> pnpm, etc.).
- [ ] Node: Build uses detected package manager (pnpm build, npm run build, etc.).
- [ ] Node: Publish always uses `npm publish --access public`.
- [ ] Bun: Build runs `bun run build`.
- [ ] Bun: Publish returns ErrNoPublish (pipeline skips publish step).
- [ ] Bun: BinaryAssets returns the compiled binary path for GitHub Release upload.
- [ ] Bun: Binary asset is uploaded to GitHub Release by the github_release step.
- [ ] Clean uses os.RemoveAll (cross-platform, no shell commands).
