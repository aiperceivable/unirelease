# UNI-008: Interactive Prompts + UX Polish

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-008                                            |
| **Phase**        | 3 - Polish                                         |
| **Priority**     | P1                                                 |
| **Effort**       | S (1-2 days)                                       |
| **Dependencies** | UNI-002                                            |
| **Packages**     | `internal/ui/`                                     |

---

## 1. Purpose

Provide colored terminal output, confirmation prompts before destructive steps, step progress display, and a summary report. Handle non-TTY environments (CI pipelines) gracefully. This is the UX layer that all other components use for user interaction.

---

## 2. Files to Create

```
internal/
  ui/
    ui.go
    ui_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/ui/ui.go

```go
package ui

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "github.com/fatih/color"
    "golang.org/x/term"
)

// UI handles all user-facing output and input.
type UI struct {
    isTTY  bool
    reader *bufio.Reader
}

// New creates a UI instance.
// Detects whether stdout is a TTY for prompt and color behavior.
func New() *UI {
    return &UI{
        isTTY:  term.IsTerminal(int(os.Stdout.Fd())),
        reader: bufio.NewReader(os.Stdin),
    }
}
```

**Color helpers:**

```go
var (
    colorRed    = color.New(color.FgRed)
    colorGreen  = color.New(color.FgGreen)
    colorYellow = color.New(color.FgYellow)
    colorBlue   = color.New(color.FgBlue)
    colorCyan   = color.New(color.FgCyan)
    colorBold   = color.New(color.Bold)
)

// Header prints a boxed header (project name + version).
// Example:
//   ╔══════════════════════════════════════════╗
//   ║  unirelease - Release Pipeline           ║
//   ╚══════════════════════════════════════════╝
func (u *UI) Header(projectType string, version string, tag string) {
    colorCyan.Println("╔══════════════════════════════════════════════════════════╗")
    colorCyan.Printf("║  unirelease - %s release v%s\n", projectType, version)
    colorCyan.Println("╚══════════════════════════════════════════════════════════╝")
    fmt.Printf("  Type:    %s\n", colorCyan.Sprint(projectType))
    fmt.Printf("  Version: %s\n", colorCyan.Sprint(version))
    fmt.Printf("  Tag:     %s\n", colorCyan.Sprint(tag))
    fmt.Println()
}
```

**Step progress:**

```go
// StepHeader prints the step progress line.
// Example: "[3/10] Verify environment..."
func (u *UI) StepHeader(current int, total int, description string) {
    colorBlue.Printf("[%d/%d] %s...\n", current, total, description)
}

// StepSkip prints a skip message.
// Example: "[skip] test (skipped by config)"
func (u *UI) StepSkip(stepName string, reason string) {
    colorYellow.Printf("[skip] %s (%s)\n", stepName, reason)
}

// StepDone prints a success message for a completed step.
func (u *UI) StepDone(message string) {
    colorGreen.Printf("  done: %s\n", message)
}
```

**Output levels:**

```go
// Info prints an informational message (cyan).
func (u *UI) Info(format string, args ...interface{}) {
    colorCyan.Printf("  "+format+"\n", args...)
}

// Warn prints a warning message (yellow).
func (u *UI) Warn(format string, args ...interface{}) {
    colorYellow.Printf("  warning: "+format+"\n", args...)
}

// Error prints an error message (red).
func (u *UI) Error(format string, args ...interface{}) {
    colorRed.Printf("  error: "+format+"\n", args...)
}

// Command prints the command being executed (dimmed).
func (u *UI) Command(cmdStr string) {
    fmt.Printf("  $ %s\n", cmdStr)
}

// DryRunMsg prints a dry-run preview message.
func (u *UI) DryRunMsg(format string, args ...interface{}) {
    colorYellow.Printf("  [dry-run] "+format+"\n", args...)
}
```

**Confirmation prompt:**

```go
// Confirm displays a yes/no prompt and returns the user's choice.
// Default is "yes" (pressing Enter accepts).
//
// Behavior:
// - TTY: displays prompt, reads input, returns true for Y/y/Enter.
// - Non-TTY (pipe/CI): returns false and prints a warning suggesting --yes.
//
// Example: "About to create git tag v1.2.3. Continue? [Y/n] "
func (u *UI) Confirm(message string) bool {
    if !u.isTTY {
        colorYellow.Printf("  %s [Y/n] (non-interactive, defaulting to no; use --yes)\n", message)
        return false
    }

    colorYellow.Printf("  %s [Y/n] ", message)
    input, err := u.reader.ReadString('\n')
    if err != nil {
        return false
    }
    input = strings.TrimSpace(strings.ToLower(input))
    // Enter (empty) or y/yes = accept
    return input == "" || input == "y" || input == "yes"
}
```

**Summary report:**

```go
// Summary prints the final release summary.
func (u *UI) Summary(results SummaryData) {
    fmt.Println()
    colorCyan.Println("╔══════════════════════════════════════════════════════════╗")
    colorCyan.Println("║  Release Summary                                        ║")
    colorCyan.Println("╚══════════════════════════════════════════════════════════╝")
    fmt.Println()
    fmt.Printf("  Version:  %s\n", colorCyan.Sprint(results.Version))
    fmt.Printf("  Tag:      %s\n", colorCyan.Sprint(results.Tag))
    fmt.Printf("  Type:     %s\n", colorCyan.Sprint(results.ProjectType))
    fmt.Println()

    for _, step := range results.Steps {
        switch step.Status {
        case StepStatusDone:
            colorGreen.Printf("  [done]  %s\n", step.Description)
        case StepStatusSkipped:
            colorYellow.Printf("  [skip]  %s\n", step.Description)
        case StepStatusFailed:
            colorRed.Printf("  [fail]  %s\n", step.Description)
        }
    }

    if results.PublishURL != "" {
        fmt.Println()
        fmt.Printf("  Published: %s\n", colorCyan.Sprint(results.PublishURL))
    }
    if results.ReleaseURL != "" {
        fmt.Printf("  Release:   %s\n", colorCyan.Sprint(results.ReleaseURL))
    }

    fmt.Println()
    colorGreen.Println("  Release complete!")
    fmt.Println()
}

// SummaryData holds the data for the final summary.
type SummaryData struct {
    Version     string
    Tag         string
    ProjectType string
    Steps       []StepResult
    PublishURL  string // e.g., "https://crates.io/crates/foo/0.3.0"
    ReleaseURL  string // e.g., "https://github.com/owner/repo/releases/tag/v0.3.0"
}

type StepStatus int

const (
    StepStatusDone    StepStatus = iota
    StepStatusSkipped
    StepStatusFailed
)

type StepResult struct {
    Name        string
    Description string
    Status      StepStatus
}
```

### 3.2 Color Behavior

The `fatih/color` library handles color support automatically:
- **TTY with color support**: Full ANSI colors.
- **Non-TTY (piped output)**: Colors automatically disabled.
- **Windows**: Uses Windows console API (no ANSI escape issues).
- **NO_COLOR env var**: Respected by `fatih/color` (disables colors).
- **FORCE_COLOR env var**: Can force colors on.

No manual detection or conditional logic needed -- `fatih/color` handles all of this.

### 3.3 Integration with Pipeline Engine

The UI instance is created in `cmd/root.go` and passed through `PipelineContext`:

```go
// In runRelease():
u := ui.New()
ctx := &pipeline.PipelineContext{
    // ...
    UI: u,
}
```

All steps use `ctx.UI` for output:
```go
// In a step:
ctx.UI.StepDone("Built successfully")
ctx.UI.Warn("Working tree has uncommitted changes")
```

The pipeline engine uses UI for step headers and skip messages:
```go
// In engine.Run():
for i, step := range e.steps {
    ctx.UI.StepHeader(i+1, len(e.steps), step.Description())
    // ...
}
```

---

## 4. Test Cases

### 4.1 UI Tests (ui_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestConfirm_TTY_EnterAccepts          | Mock TTY, input "\n"                          | true                                |
| TestConfirm_TTY_YAccepts              | Mock TTY, input "y\n"                         | true                                |
| TestConfirm_TTY_NDeclines             | Mock TTY, input "n\n"                         | false                               |
| TestConfirm_TTY_YesAccepts            | Mock TTY, input "yes\n"                       | true                                |
| TestConfirm_NonTTY_ReturnsFalse       | isTTY=false                                   | false (with warning message)        |
| TestStepHeader_Format                  | StepHeader(3, 10, "Build")                    | Output contains "[3/10] Build..."   |
| TestStepSkip_Format                    | StepSkip("test", "skipped by config")         | Output contains "[skip] test"       |
| TestDryRunMsg_Format                   | DryRunMsg("Would run: cargo build")           | Output contains "[dry-run]"         |
| TestSummary_AllDone                    | All steps StepStatusDone                      | Output contains "[done]" for each   |
| TestSummary_WithSkips                  | Some steps skipped                            | Output contains "[skip]" and "[done]" |
| TestSummary_WithPublishURL             | PublishURL set                                | Output contains URL                 |

---

## 5. Output Examples

### 5.1 Full Pipeline Run

```
╔══════════════════════════════════════════════════════════╗
║  unirelease - rust release v0.3.0
╚══════════════════════════════════════════════════════════╝
  Type:    rust
  Version: 0.3.0
  Tag:     v0.3.0

[1/10] Detect project type...
  done: Detected rust (from Cargo.toml)
[2/10] Read version...
  done: Version 0.3.0 from Cargo.toml
[3/10] Verify environment...
  done: cargo 1.77.0, rustc 1.77.0
[4/10] Check git status...
  done: Clean working tree, branch: main
[5/10] Clean build artifacts...
  $ cargo clean
  done: Cleaned
[6/10] Build project...
  $ cargo build --release
  done: Built successfully
[7/10] Run tests...
  $ cargo test
  done: All tests passed
[8/10] Create git tag...
  About to create and push tag v0.3.0. Continue? [Y/n] y
  $ git tag -a v0.3.0 -m "Release 0.3.0"
  $ git push origin v0.3.0
  done: Tag v0.3.0 pushed
[9/10] Create GitHub Release...
  About to create GitHub Release v0.3.0. Continue? [Y/n] y
  done: Release created
[10/10] Publish to crates.io...
  About to publish to crates.io. Continue? [Y/n] y
  $ cargo publish
  done: Published to crates.io

╔══════════════════════════════════════════════════════════╗
║  Release Summary                                        ║
╚══════════════════════════════════════════════════════════╝

  Version:  0.3.0
  Tag:      v0.3.0
  Type:     rust

  [done]  Detect project type
  [done]  Read version
  [done]  Verify environment
  [done]  Check git status
  [done]  Clean build artifacts
  [done]  Build project
  [done]  Run tests
  [done]  Create git tag
  [done]  Create GitHub Release
  [done]  Publish to crates.io

  Published: https://crates.io/crates/myproject/0.3.0
  Release:   https://github.com/owner/repo/releases/tag/v0.3.0

  Release complete!
```

### 5.2 Dry Run

```
╔══════════════════════════════════════════════════════════╗
║  unirelease - rust release v0.3.0 (DRY RUN)
╚══════════════════════════════════════════════════════════╝

[1/10] Detect project type...
  [dry-run] Detected rust (from Cargo.toml)
[2/10] Read version...
  [dry-run] Version 0.3.0 from Cargo.toml
...
[6/10] Build project...
  [dry-run] Would run: cargo build --release
...
[10/10] Publish to crates.io...
  [dry-run] Would run: cargo publish
```

---

## 6. Acceptance Criteria

- [ ] Output is colored on terminals that support it.
- [ ] Colors are automatically disabled when output is piped (non-TTY).
- [ ] `NO_COLOR` environment variable disables colors.
- [ ] Step progress shows `[N/10]` format with step description.
- [ ] Skipped steps show `[skip]` with the reason.
- [ ] Destructive steps prompt `[Y/n]` when `--yes` is not set.
- [ ] Enter (empty input) accepts the prompt (default yes).
- [ ] Non-TTY environments default to "no" on prompts, with a warning about `--yes`.
- [ ] `--yes` flag suppresses all prompts.
- [ ] Summary report shows all steps with their status (done/skip/fail).
- [ ] Summary includes publish URL and release URL when available.
- [ ] Dry-run output clearly prefixes every action with `[dry-run]`.
- [ ] Output works correctly on macOS, Linux, and Windows terminals.
