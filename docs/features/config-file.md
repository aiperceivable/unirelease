# UNI-007: Config File Support (.unirelease.toml)

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-007                                            |
| **Phase**        | 3 - Polish                                         |
| **Priority**     | P1                                                 |
| **Effort**       | S (1-2 days)                                       |
| **Dependencies** | UNI-002                                            |
| **Packages**     | `internal/config/`                                 |

---

## 1. Purpose

Parse an optional `.unirelease.toml` file from the project root to allow overriding default behaviors: project type, tag prefix, step skipping, pre/post hooks, and custom build/test/clean commands. The config file is entirely optional -- unirelease works with zero configuration for standard project layouts.

---

## 2. Files to Create

```
internal/
  config/
    config.go
    config_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/config/config.go

**Config structs:**

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/BurntSushi/toml"
)

// Config represents the merged configuration.
type Config struct {
    Type      string         `toml:"type"`
    TagPrefix string         `toml:"tag_prefix"`
    Skip      []string       `toml:"skip"`
    Hooks     HooksConfig    `toml:"hooks"`
    Commands  CommandsConfig `toml:"commands"`
}

type HooksConfig struct {
    PreBuild    string `toml:"pre_build"`
    PostBuild   string `toml:"post_build"`
    PrePublish  string `toml:"pre_publish"`
    PostPublish string `toml:"post_publish"`
}

type CommandsConfig struct {
    Build string `toml:"build"`
    Test  string `toml:"test"`
    Clean string `toml:"clean"`
}
```

**Default function:**

```go
// Default returns a Config with default values.
func Default() *Config {
    return &Config{
        Type:      "",  // empty = auto-detect
        TagPrefix: "v", // default: v1.2.3
        Skip:      nil,
        Hooks:     HooksConfig{},
        Commands:  CommandsConfig{},
    }
}
```

**Load function:**

```go
// Load reads .unirelease.toml from the project directory.
// If the file does not exist, returns Default() with no error.
// If the file exists but is malformed, returns an error with line/column info.
func Load(projectDir string) (*Config, error) {
    path := filepath.Join(projectDir, ".unirelease.toml")

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return Default(), nil // No config file = use defaults
        }
        return nil, fmt.Errorf("read config: %w", err)
    }

    cfg := Default()
    if err := toml.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parse .unirelease.toml: %w", err)
    }

    // Validate
    if err := cfg.validate(); err != nil {
        return nil, fmt.Errorf(".unirelease.toml: %w", err)
    }

    return cfg, nil
}
```

**Validate function:**

```go
// validate checks the config for invalid values.
func (c *Config) validate() error {
    // Validate type override
    if c.Type != "" {
        valid := map[string]bool{"rust": true, "node": true, "bun": true, "python": true}
        if !valid[c.Type] {
            return fmt.Errorf("invalid type %q; must be one of: rust, node, bun, python", c.Type)
        }
    }

    // Validate skip step names
    validSteps := map[string]bool{
        "detect": true, "read_version": true, "verify_env": true,
        "check_git_status": true, "clean": true, "build": true,
        "test": true, "git_tag": true, "github_release": true, "publish": true,
    }
    for _, step := range c.Skip {
        if !validSteps[step] {
            return fmt.Errorf("invalid skip step %q; valid steps: %v", step, validStepNames())
        }
    }

    return nil
}

func validStepNames() []string {
    return []string{
        "detect", "read_version", "verify_env", "check_git_status",
        "clean", "build", "test", "git_tag", "github_release", "publish",
    }
}
```

**Merge function (CLI flags take precedence):**

```go
// Merge applies CLI flag overrides to the config.
// CLI flags always take precedence over config file values.
func (c *Config) Merge(cliType string, cliVersion string) {
    if cliType != "" {
        c.Type = cliType // --type flag overrides config type
    }
    // Note: cliVersion is not stored in Config -- it is handled
    // by PipelineContext directly. This method only handles
    // fields that exist in both config and CLI.
}
```

**HasSkip helper:**

```go
// HasSkip checks if a step name is in the skip list.
func (c *Config) HasSkip(stepName string) bool {
    for _, s := range c.Skip {
        if s == stepName {
            return true
        }
    }
    return false
}
```

### 3.2 Hook Execution (in pipeline engine)

The pipeline engine (UNI-002) already has hook execution points. This feature provides the config values that feed into those points. Hook commands are executed via the Runner:

```go
// executeHook runs a hook command if non-empty.
// Hooks run through the system shell for maximum flexibility:
//   Unix: sh -c "<command>"
//   Windows: cmd /c "<command>"
func executeHook(ctx *PipelineContext, hookCmd string, hookName string) error {
    if hookCmd == "" {
        return nil
    }
    ctx.UI.Info("Running %s hook: %s", hookName, hookCmd)
    if runtime.GOOS == "windows" {
        _, err := ctx.Runner.Run("cmd", "/c", hookCmd)
        return err
    }
    _, err := ctx.Runner.Run("sh", "-c", hookCmd)
    return err
}
```

### 3.3 Command Override Logic (in pipeline steps)

When `Config.Commands.Build`, `.Test`, or `.Clean` is non-empty, the corresponding step uses the override command instead of the provider's method:

```go
// In BuildStep.Execute():
func (s *BuildStep) Execute(ctx *PipelineContext) error {
    if ctx.Config != nil && ctx.Config.Commands.Build != "" {
        ctx.UI.Info("Using custom build command: %s", ctx.Config.Commands.Build)
        if runtime.GOOS == "windows" {
            _, err := ctx.Runner.Run("cmd", "/c", ctx.Config.Commands.Build)
            return err
        }
        _, err := ctx.Runner.Run("sh", "-c", ctx.Config.Commands.Build)
        return err
    }
    return ctx.Provider.Build(ctx)
}
```

Same pattern for TestStep and CleanStep.

---

## 4. Test Cases

### 4.1 Config Tests (config_test.go)

| Test Name                              | Input                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestLoad_NoFile                        | Dir without .unirelease.toml                  | Default config, no error            |
| TestLoad_EmptyFile                     | Empty .unirelease.toml                        | Default config, no error            |
| TestLoad_TypeOnly                      | `type = "rust"`                               | Type="rust", rest defaults          |
| TestLoad_FullConfig                    | All fields set                                | All fields populated                |
| TestLoad_InvalidType                   | `type = "java"`                               | Error: invalid type                 |
| TestLoad_InvalidSkipStep               | `skip = ["bogus"]`                            | Error: invalid skip step            |
| TestLoad_MalformedToml                 | Invalid TOML syntax                           | Parse error                          |
| TestLoad_TagPrefix                     | `tag_prefix = "release/v"`                    | TagPrefix = "release/v"             |
| TestLoad_Hooks                         | `[hooks]\npre_build = "make gen"`             | Hooks.PreBuild = "make gen"         |
| TestLoad_Commands                      | `[commands]\nbuild = "make release"`          | Commands.Build = "make release"     |
| TestDefault_Values                     | None                                          | TagPrefix="v", Type="", Skip=nil    |
| TestMerge_CLIOverridesConfig           | Config Type="rust", CLI Type="node"           | Type="node"                         |
| TestMerge_CLIEmpty_KeepsConfig         | Config Type="rust", CLI Type=""               | Type="rust"                         |
| TestHasSkip_True                       | Skip=["test","clean"], check "test"           | true                                |
| TestHasSkip_False                      | Skip=["test","clean"], check "build"          | false                               |

---

## 5. Config File Reference

```toml
# .unirelease.toml -- all fields are optional

# Override auto-detection. Values: rust, node, bun, python
type = "rust"

# Tag prefix prepended to version. Default: "v" (-> v1.2.3)
tag_prefix = "v"

# Steps to skip. Any step name is valid:
# detect, read_version, verify_env, check_git_status, clean, build, test,
# git_tag, github_release, publish
skip = ["clean", "test"]

[hooks]
pre_build = "make generate"    # Runs before build step
post_build = ""                # Runs after build step
pre_publish = ""               # Runs before publish step
post_publish = "notify.sh"    # Runs after publish step

[commands]
build = "make release"         # Overrides provider's default build command
test = "make test-release"     # Overrides provider's default test command
clean = "make clean"           # Overrides provider's default clean command
```

---

## 6. Acceptance Criteria

- [ ] `.unirelease.toml` is loaded from the project root when present.
- [ ] Missing `.unirelease.toml` is not an error -- defaults are used.
- [ ] `type = "rust"` overrides auto-detection to Rust.
- [ ] `tag_prefix = "release/"` produces tags like `release/1.2.3`.
- [ ] `skip = ["test"]` causes the test step to be skipped with a "[skip]" message.
- [ ] Invalid type value produces a clear error with valid options.
- [ ] Invalid step name in skip produces a clear error with valid step names.
- [ ] Malformed TOML produces a parse error (not a silent failure).
- [ ] `[hooks] pre_build = "make gen"` runs `make gen` before the build step.
- [ ] Hook failure stops the pipeline.
- [ ] `[commands] build = "make release"` replaces the provider's build command.
- [ ] CLI `--type` flag overrides config file `type` value.
- [ ] Hooks execute through the system shell (sh -c on Unix, cmd /c on Windows).
