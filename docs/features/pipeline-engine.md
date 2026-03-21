# UNI-002: Pipeline Engine

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-002                                            |
| **Phase**        | 1 - Core Pipeline                                  |
| **Priority**     | P0                                                 |
| **Effort**       | M (3-5 days)                                       |
| **Dependencies** | UNI-001                                            |
| **Packages**     | `internal/pipeline/`, `internal/runner/`           |

---

## 1. Purpose

Orchestrate the 10-step release pipeline, handling step sequencing, `--step` filtering, `--dry-run` preview, step skipping (from config), hook execution, and error propagation. This is the central coordination point that all providers and features plug into.

---

## 2. Files to Create

```
internal/
  pipeline/
    engine.go
    engine_test.go
    step.go
    context.go
    steps/
      detect.go
      version.go
      verify_env.go
      git_status.go
      clean.go
      build.go
      test.go
      git_tag.go
      github_release.go
      publish.go
  runner/
    runner.go
    runner_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/pipeline/step.go

```go
package pipeline

// Step represents a single pipeline step.
type Step interface {
    Name() string
    Description() string
    Execute(ctx *PipelineContext) error
    DryRun(ctx *PipelineContext) error
    Destructive() bool
}

// StepNames defines the canonical order and valid step names.
var StepNames = []string{
    "detect",
    "read_version",
    "verify_env",
    "check_git_status",
    "clean",
    "build",
    "test",
    "git_tag",
    "github_release",
    "publish",
}

// ValidStepName checks if a name is a recognized pipeline step.
func ValidStepName(name string) bool {
    for _, n := range StepNames {
        if n == name {
            return true
        }
    }
    return false
}
```

### 3.2 internal/pipeline/context.go

```go
package pipeline

import (
    "unirelease/internal/config"
    "unirelease/internal/providers"
    "unirelease/internal/runner"
    "unirelease/internal/ui"
)

// PipelineContext carries shared state through the pipeline.
type PipelineContext struct {
    ProjectDir  string
    ProjectType string
    Version     string
    TagName     string
    Provider    providers.Provider
    Config      *config.Config
    DryRun      bool
    Yes         bool
    Step        string // if non-empty, run only this step
    Runner      *runner.Runner
    UI          *ui.UI
    GitHubRepo  string
    GitHubToken string
}

// FormatTag applies the tag prefix to the version.
// Default prefix is "v", producing "v1.2.3".
func (ctx *PipelineContext) FormatTag() string {
    prefix := "v"
    if ctx.Config != nil && ctx.Config.TagPrefix != "" {
        prefix = ctx.Config.TagPrefix
    }
    return prefix + ctx.Version
}
```

### 3.3 internal/pipeline/engine.go

```go
package pipeline

// Engine orchestrates pipeline execution.
type Engine struct {
    steps []Step
    ctx   *PipelineContext
}

// NewEngine creates an engine with the default step order.
func NewEngine(ctx *PipelineContext) *Engine

// Run executes the pipeline.
func (e *Engine) Run() error
```

**Run() logic (step by step):**

1. If `ctx.Step` is non-empty (single step mode):
   - Find the step with matching `Name()`.
   - If not found, return `fmt.Errorf("unknown step: %q, valid steps: %v", ctx.Step, StepNames)`.
   - Execute that single step (see step 5 below).
   - Return.

2. For full pipeline mode, iterate over `e.steps` in order:

3. For each step, check skip conditions:
   - If step name is in `ctx.Config.Skip`, print `"[skip] <step name> (skipped by config)"` and continue.
   - (No other skip conditions checked by the engine -- individual steps handle their own "not applicable" logic.)

4. Check for pre/post hooks:
   - Before "build" step: if `ctx.Config.Hooks.PreBuild` is non-empty, execute it via `ctx.Runner.Run("sh", "-c", hookCmd)` (on Windows: `cmd /c`).
   - After "build" step: if `ctx.Config.Hooks.PostBuild` is non-empty, execute it.
   - Before "publish" step: if `ctx.Config.Hooks.PrePublish` is non-empty, execute it.
   - After "publish" step: if `ctx.Config.Hooks.PostPublish` is non-empty, execute it.
   - Hook failure is treated as a step failure (pipeline stops).

5. Execute the step:
   - Print step header: `"[N/10] <step description>..."` (where N is the step number).
   - If `ctx.DryRun`, call `step.DryRun(ctx)`.
   - Else if `step.Destructive()` and not `ctx.Yes`:
     - Call `ctx.UI.Confirm(fmt.Sprintf("About to %s. Continue?", step.Description()))`.
     - If user declines, print `"[skip] <step name> (user declined)"` and continue.
   - Else, call `step.Execute(ctx)`.
   - If error returned:
     - If error is `ErrNoPublish`, print `"[skip] <step name> (not applicable)"` and continue.
     - Otherwise, return `fmt.Errorf("step %s: %w", step.Name(), err)`.

6. After all steps complete, print summary (step count, version, tag).

### 3.4 Pipeline Steps (internal/pipeline/steps/)

Each step file implements the `Step` interface. All step files follow this pattern:

```go
package steps

import "unirelease/internal/pipeline"

type BuildStep struct{}

func (s *BuildStep) Name() string        { return "build" }
func (s *BuildStep) Description() string  { return "Build project" }
func (s *BuildStep) Destructive() bool    { return false }

func (s *BuildStep) Execute(ctx *pipeline.PipelineContext) error {
    // Check if config overrides the build command
    if ctx.Config != nil && ctx.Config.Commands.Build != "" {
        _, err := ctx.Runner.Run("sh", "-c", ctx.Config.Commands.Build)
        return err
    }
    return ctx.Provider.Build(ctx)
}

func (s *BuildStep) DryRun(ctx *pipeline.PipelineContext) error {
    ctx.UI.DryRunMsg("Would build project using %s provider", ctx.Provider.Name())
    return nil
}
```

**Step implementations summary:**

| Step File         | Name             | Destructive | Delegates To                    |
|-------------------|------------------|-------------|----------------------------------|
| detect.go         | detect           | false       | `detector.Detect()`              |
| version.go        | read_version     | false       | `detector.ReadVersion()`         |
| verify_env.go     | verify_env       | false       | `provider.VerifyEnv()`           |
| git_status.go     | check_git_status | false       | `git.Status()`, `git.CurrentBranch()` |
| clean.go          | clean            | false       | `provider.Clean()`               |
| build.go          | build            | false       | `provider.Build()` or config override |
| test.go           | test             | false       | `provider.Test()` or config override |
| git_tag.go        | git_tag          | true        | `git.CreateTag()`, `git.PushTag()` |
| github_release.go | github_release   | true        | `github.CreateRelease()`, `github.UploadAsset()` |
| publish.go        | publish          | true        | `provider.Publish()`             |

**detect.go Execute() logic:**
1. Call `detector.Detect(ctx.ProjectDir, ctx.Config.Type)` -- type override from config or CLI flag.
2. Set `ctx.ProjectType = result.Type`.
3. Resolve provider: call `providers.ForType(result.Type)`.
4. Set `ctx.Provider = provider`.
5. Print: `"Detected project type: <type> (from <manifest>)"`.

**version.go Execute() logic:**
1. Call `detector.ReadVersion(ctx.ProjectDir, ctx.ProjectType, versionOverride)`.
2. Set `ctx.Version = version`.
3. Set `ctx.TagName = ctx.FormatTag()`.
4. Print: `"Version: <version>, Tag: <tag>"`.

**verify_env.go Execute() logic:**
1. Call `ctx.Provider.VerifyEnv()`.
2. If missing tools returned, format as: `"Missing required tools: cargo, rustc"`.
3. Return `ErrMissingTool` with the formatted message.

**check_git_status.go Execute() logic:**
1. Call `git.Status(ctx.ProjectDir)`.
2. If not clean, print warning with `git status --short` output.
3. If not `ctx.Yes`, prompt: `"Working tree has uncommitted changes. Continue?"`.
4. If declined, return error.
5. Call `git.CurrentBranch(ctx.ProjectDir)`.
6. Print: `"Branch: <branch>"`.

**git_tag.go Execute() logic:**
1. Check if tag exists locally: `git.TagExists(ctx.ProjectDir, ctx.TagName)`.
2. Check if tag exists on remote: `git.TagExistsOnRemote(ctx.ProjectDir, ctx.TagName)`.
3. If exists on remote, print warning, return (not an error -- tag already done).
4. If exists locally but not remote, push it: `git.PushTag(ctx.ProjectDir, ctx.TagName)`.
5. Otherwise, create tag: `git.CreateTag(ctx.ProjectDir, ctx.TagName, "Release "+ctx.Version)`.
6. Push tag: `git.PushTag(ctx.ProjectDir, ctx.TagName)`.

**github_release.go Execute() logic:**
1. Resolve GitHub repo: `git.ParseGitHubRepo(git.RemoteURL(ctx.ProjectDir))`.
2. Set `ctx.GitHubRepo`.
3. Resolve token: `github.ResolveToken()`.
4. If no token, print instructions (3 options), skip step (not fatal).
5. Check if release exists: `client.ReleaseExists(ctx.TagName)`.
6. If exists, print "Release already exists", skip.
7. Create release: `client.CreateRelease(ctx.TagName, "Release "+ctx.Version, "Release version "+ctx.Version)`.
8. Check `ctx.Provider.BinaryAssets(ctx)` -- if non-nil, upload each asset.

**publish.go Execute() logic:**
1. Call `ctx.Provider.Publish(ctx)`.
2. If error is `ErrNoPublish`, return it (engine treats as skip).
3. Print: `"Published to <provider.PublishTarget()>"`.

### 3.5 internal/runner/runner.go

```go
package runner

import (
    "fmt"
    "os/exec"
    "strings"

    "unirelease/internal/ui"
)

type Runner struct {
    DryRun bool
    Dir    string
    UI     *ui.UI
}

func New(dir string, dryRun bool, u *ui.UI) *Runner {
    return &Runner{DryRun: dryRun, Dir: dir, UI: u}
}

// Run executes a command, printing it first.
// In dry-run mode, prints "[dry-run] Would run: <cmd>" and returns ("", nil).
func (r *Runner) Run(name string, args ...string) (string, error) {
    cmdStr := name + " " + strings.Join(args, " ")

    if r.DryRun {
        r.UI.DryRunMsg("Would run: %s", cmdStr)
        return "", nil
    }

    r.UI.Command(cmdStr)
    cmd := exec.Command(name, args...)
    cmd.Dir = r.Dir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    err := cmd.Run()
    if err != nil {
        return "", fmt.Errorf("command failed: %s: %w", cmdStr, err)
    }
    return "", nil
}

// RunSilent executes a command and captures output (no printing).
func (r *Runner) RunSilent(name string, args ...string) (string, error) {
    if r.DryRun {
        return "", nil
    }
    cmd := exec.Command(name, args...)
    cmd.Dir = r.Dir
    out, err := cmd.CombinedOutput()
    return strings.TrimSpace(string(out)), err
}

// CommandExists checks if an executable is in PATH.
func (r *Runner) CommandExists(name string) bool {
    _, err := exec.LookPath(name)
    return err == nil
}
```

---

## 4. Test Cases

### 4.1 Engine Tests (engine_test.go)

| Test Name                              | Setup                                         | Expected Behavior                   |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestEngine_FullPipeline                | Mock all 10 steps, mock provider              | All steps executed in order         |
| TestEngine_SingleStep                  | ctx.Step = "build"                            | Only BuildStep.Execute called       |
| TestEngine_SingleStep_Invalid          | ctx.Step = "bogus"                            | Error: unknown step                 |
| TestEngine_SkipConfig                  | Config.Skip = ["test", "clean"]               | test and clean steps skipped        |
| TestEngine_DryRun                      | ctx.DryRun = true                             | DryRun() called on each step, not Execute() |
| TestEngine_DestructivePrompt_Yes       | Destructive step, ctx.Yes = false, UI returns true | Execute() called              |
| TestEngine_DestructivePrompt_Decline   | Destructive step, ctx.Yes = false, UI returns false | Step skipped                 |
| TestEngine_DestructiveNoPrompt_YesFlag | Destructive step, ctx.Yes = true              | Execute() called without prompt     |
| TestEngine_ErrNoPublish_SkipsStep      | Publish step returns ErrNoPublish             | Step skipped, no error              |
| TestEngine_StepError_StopsPipeline     | Build step returns error                      | Pipeline stops, error propagated    |
| TestEngine_PreBuildHook                | Config.Hooks.PreBuild = "echo hello"          | Hook runs before build step         |
| TestEngine_HookFailure_StopsPipeline   | Config.Hooks.PreBuild = "false"               | Pipeline stops with hook error      |

### 4.2 Runner Tests (runner_test.go)

| Test Name                              | Setup                                         | Expected Behavior                   |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestRunner_Run_Success                 | Run "echo" "hello"                            | No error, command executes          |
| TestRunner_Run_Failure                 | Run "false"                                   | Error returned                      |
| TestRunner_Run_DryRun                  | DryRun=true, Run "anything"                   | No command executed, no error       |
| TestRunner_RunSilent_CapturesOutput    | RunSilent "echo" "hello"                      | Returns "hello"                     |
| TestRunner_CommandExists_True          | CommandExists "go"                            | true                                |
| TestRunner_CommandExists_False         | CommandExists "nonexistent_binary_xyz"        | false                               |

---

## 5. Acceptance Criteria

- [ ] Full pipeline runs all 10 steps in the defined order.
- [ ] `--step build` runs only the build step and exits.
- [ ] `--step invalid_name` prints valid step names and exits with code 2.
- [ ] `--dry-run` calls DryRun() on every step, never Execute().
- [ ] Steps in `Config.Skip` are skipped with a "[skip]" message.
- [ ] Destructive steps prompt for confirmation when `--yes` is not set.
- [ ] `--yes` bypasses all confirmation prompts.
- [ ] `ErrNoPublish` from a provider causes the publish step to skip (not fail).
- [ ] A step error stops the pipeline and propagates the error with step context.
- [ ] Pre/post hooks execute at the correct points in the pipeline.
- [ ] Hook failure stops the pipeline.
- [ ] Runner in dry-run mode prints commands but does not execute them.
