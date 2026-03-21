package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aipartnerup/unirelease/internal/config"
	"github.com/aipartnerup/unirelease/internal/detector"
	"github.com/aipartnerup/unirelease/internal/pipeline"
	"github.com/aipartnerup/unirelease/internal/pipeline/steps"
	"github.com/aipartnerup/unirelease/internal/providers"
	"github.com/aipartnerup/unirelease/internal/runner"
	"github.com/aipartnerup/unirelease/internal/ui"
)

func newTestUI() *ui.UI {
	reader := bufio.NewReader(strings.NewReader(""))
	return ui.NewWithReader(reader, false)
}

// setupE2EProject creates a temp directory with project files and a git repo.
func setupE2EProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for name, content := range files {
		path := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	gitExec(t, dir, "init", dir)
	gitExec(t, dir, "-C", dir, "config", "user.email", "test@test.com")
	gitExec(t, dir, "-C", dir, "config", "user.name", "Test")
	gitExec(t, dir, "-C", dir, "add", ".")
	gitExec(t, dir, "-C", dir, "commit", "-m", "initial")

	return dir
}

func gitExec(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

// detectAndResolveStep wraps DetectStep to also resolve the provider.
type detectAndResolveStep struct {
	inner *steps.DetectStep
}

func (s *detectAndResolveStep) Name() string        { return s.inner.Name() }
func (s *detectAndResolveStep) Description() string { return s.inner.Description() }
func (s *detectAndResolveStep) Destructive() bool   { return s.inner.Destructive() }

func (s *detectAndResolveStep) Execute(ctx *pipeline.Context) error {
	if err := s.inner.Execute(ctx); err != nil {
		return err
	}
	p, err := providers.ForType(ctx.ProjectType)
	if err != nil {
		return err
	}
	ctx.Provider = p
	return nil
}

func (s *detectAndResolveStep) DryRun(ctx *pipeline.Context) error {
	if err := s.inner.DryRun(ctx); err != nil {
		return err
	}
	p, err := providers.ForType(ctx.ProjectType)
	if err != nil {
		return err
	}
	ctx.Provider = p
	return nil
}

func buildE2EPipeline(t *testing.T, dir string) (*pipeline.Engine, *pipeline.Context) {
	t.Helper()

	u := newTestUI()
	r := runner.New(dir, true, u) // dry-run
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ctx := &pipeline.Context{
		ProjectDir: dir,
		Config:     cfg,
		DryRun:     true,
		Yes:        true,
		Runner:     r,
		UI:         u,
	}

	allSteps := []pipeline.Step{
		&detectAndResolveStep{inner: &steps.DetectStep{}},
		&steps.ReadVersionStep{},
		&steps.VerifyEnvStep{},
		&steps.CheckGitStatusStep{},
		&steps.CleanStep{},
		&steps.BuildStep{},
		&steps.TestStep{},
		&steps.VerifyStep{},
		&steps.GitTagStep{},
		&steps.GitHubReleaseStep{},
		&steps.PublishStep{},
	}

	engine := pipeline.NewEngine(ctx, allSteps)
	return engine, ctx
}

// --- E2E: Go Project ---

func TestE2E_DryRun_GoProject(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"go.mod":  "module github.com/test/myapp\n\ngo 1.22\n",
		"main.go": "package main\nfunc main() {}\n",
		"VERSION": "1.0.0",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.ProjectType != "go" {
		t.Errorf("expected 'go', got %q", ctx.ProjectType)
	}
	if ctx.Version != "1.0.0" {
		t.Errorf("expected '1.0.0', got %q", ctx.Version)
	}
	if ctx.TagName != "v1.0.0" {
		t.Errorf("expected 'v1.0.0', got %q", ctx.TagName)
	}
	if ctx.Provider == nil {
		t.Fatal("expected provider to be set")
	}
	if ctx.Provider.Name() != "go" {
		t.Errorf("expected provider 'go', got %q", ctx.Provider.Name())
	}
}

// --- E2E: Rust Project ---

func TestE2E_DryRun_RustProject(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"Cargo.toml":  "[package]\nname = \"myapp\"\nversion = \"2.5.0\"\nedition = \"2021\"\n",
		"src/main.rs": "fn main() {}\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.ProjectType != "rust" {
		t.Errorf("expected 'rust', got %q", ctx.ProjectType)
	}
	if ctx.Version != "2.5.0" {
		t.Errorf("expected '2.5.0', got %q", ctx.Version)
	}
	if ctx.TagName != "v2.5.0" {
		t.Errorf("expected 'v2.5.0', got %q", ctx.TagName)
	}
}

// --- E2E: Node Project ---

func TestE2E_DryRun_NodeProject(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"package.json": `{"name": "myapp", "version": "3.0.0", "scripts": {"build": "tsc", "test": "jest"}}`,
		"index.js":     "console.log('hello')\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.ProjectType != "node" {
		t.Errorf("expected 'node', got %q", ctx.ProjectType)
	}
	if ctx.Version != "3.0.0" {
		t.Errorf("expected '3.0.0', got %q", ctx.Version)
	}
}

// --- E2E: Python Project ---

func TestE2E_DryRun_PythonProject(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"pyproject.toml":        "[project]\nname = \"myapp\"\nversion = \"0.1.0\"\n",
		"src/myapp/__init__.py": "__version__ = '0.1.0'\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.ProjectType != "python" {
		t.Errorf("expected 'python', got %q", ctx.ProjectType)
	}
	if ctx.Version != "0.1.0" {
		t.Errorf("expected '0.1.0', got %q", ctx.Version)
	}
}

// --- E2E: Bun Binary Project ---

func TestE2E_DryRun_BunProject(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"package.json": `{"name": "myapp", "version": "1.0.0", "scripts": {"build": "bun build --compile src/index.ts --outfile dist/myapp"}}`,
		"src/index.ts": "console.log('hello')\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.ProjectType != "bun" {
		t.Errorf("expected 'bun', got %q", ctx.ProjectType)
	}
}

// --- E2E: Config Overrides ---

func TestE2E_DryRun_WithConfig(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"go.mod":  "module github.com/test/myapp\n\ngo 1.22\n",
		"main.go": "package main\nfunc main() {}\n",
		"VERSION": "1.0.0",
		".unirelease.toml": "tag_prefix = \"go/v\"\nskip = [\"verify\"]\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}

	if ctx.TagName != "go/v1.0.0" {
		t.Errorf("expected tag 'go/v1.0.0', got %q", ctx.TagName)
	}
}

// --- E2E: CHANGELOG ---

func TestE2E_DryRun_WithChangelog(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"go.mod":  "module github.com/test/myapp\n\ngo 1.22\n",
		"main.go": "package main\nfunc main() {}\n",
		"VERSION": "1.0.0",
		"CHANGELOG.md": "# Changelog\n\n## [1.0.0] - 2026-03-21\n\n### Added\n- First stable release\n\n## [0.9.0]\n\n- Beta\n",
	})

	engine, ctx := buildE2EPipeline(t, dir)
	if err := engine.Run(); err != nil {
		t.Fatalf("E2E dry-run failed: %v", err)
	}
	if ctx.Version != "1.0.0" {
		t.Errorf("expected '1.0.0', got %q", ctx.Version)
	}
}

// --- E2E: Detection Priority ---

func TestE2E_DetectionPriority_GoOverNode(t *testing.T) {
	dir := setupE2EProject(t, map[string]string{
		"go.mod":       "module github.com/test/myapp\n\ngo 1.22\n",
		"package.json": `{"name": "myapp", "version": "1.0.0"}`,
		"VERSION":      "1.0.0",
	})

	result, err := detector.Detect(dir, "")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if result.Type != detector.TypeGo {
		t.Errorf("expected Go over Node, got %s", result.Type)
	}
}
