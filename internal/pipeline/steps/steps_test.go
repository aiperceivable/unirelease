package steps

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/pipeline"
	"github.com/aiperceivable/unirelease/internal/providers"
	"github.com/aiperceivable/unirelease/internal/runner"
	"github.com/aiperceivable/unirelease/internal/ui"
)

func newTestUI() *ui.UI {
	reader := bufio.NewReader(strings.NewReader(""))
	return ui.NewWithReader(reader, false)
}

func newTestContext(dir string, dryRun bool) *pipeline.Context {
	u := newTestUI()
	r := runner.New(dir, dryRun, u)
	return &pipeline.Context{
		ProjectDir: dir,
		Config:     config.Default(),
		DryRun:     dryRun,
		Yes:        true,
		Runner:     r,
		UI:         u,
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s: %v", args, string(out), err)
	}
}

func setupGitProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", dir)
	gitRun(t, dir, "-C", dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "-C", dir, "config", "user.name", "Test")
	return dir
}

// --- DetectStep ---

func TestDetectStep_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.1.0\"\n")

	ctx := newTestContext(dir, false)
	step := &DetectStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ProjectType != "rust" {
		t.Errorf("expected 'rust', got %q", ctx.ProjectType)
	}
}

func TestDetectStep_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")

	ctx := newTestContext(dir, false)
	step := &DetectStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ProjectType != "go" {
		t.Errorf("expected 'go', got %q", ctx.ProjectType)
	}
}

func TestDetectStep_Override(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.1.0\"\n")

	ctx := newTestContext(dir, false)
	ctx.TypeOverride = "python"
	step := &DetectStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ProjectType != "python" {
		t.Errorf("expected 'python' override, got %q", ctx.ProjectType)
	}
}

func TestDetectStep_NoManifest(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, false)
	step := &DetectStep{}
	if err := step.Execute(ctx); err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestDetectStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")

	ctx := newTestContext(dir, true)
	step := &DetectStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.ProjectType != "go" {
		t.Errorf("expected 'go', got %q", ctx.ProjectType)
	}
}

// --- ReadVersionStep ---

func TestReadVersionStep_Cargo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"1.2.3\"\n")

	ctx := newTestContext(dir, false)
	ctx.ProjectType = "rust"
	step := &ReadVersionStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Version != "1.2.3" {
		t.Errorf("expected '1.2.3', got %q", ctx.Version)
	}
	if ctx.TagName != "v1.2.3" {
		t.Errorf("expected 'v1.2.3', got %q", ctx.TagName)
	}
}

func TestReadVersionStep_GoVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "2.0.0")

	ctx := newTestContext(dir, false)
	ctx.ProjectType = "go"
	step := &ReadVersionStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Version != "2.0.0" {
		t.Errorf("expected '2.0.0', got %q", ctx.Version)
	}
}

func TestReadVersionStep_Override(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.1.0\"\n")

	ctx := newTestContext(dir, false)
	ctx.ProjectType = "rust"
	ctx.VersionOverride = "9.9.9"
	step := &ReadVersionStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.Version != "9.9.9" {
		t.Errorf("expected '9.9.9', got %q", ctx.Version)
	}
}

// --- VerifyEnvStep ---

func TestVerifyEnvStep_NoProvider(t *testing.T) {
	ctx := newTestContext(t.TempDir(), false)
	step := &VerifyEnvStep{}
	if err := step.Execute(ctx); err == nil {
		t.Fatal("expected error when no provider set")
	}
}

func TestVerifyEnvStep_WithGoProvider(t *testing.T) {
	ctx := newTestContext(t.TempDir(), false)
	p, _ := providers.ForType("go")
	ctx.Provider = p
	ctx.ProjectType = "go"

	step := &VerifyEnvStep{}
	err := step.Execute(ctx)
	// go should be available in test environment
	if err != nil {
		t.Fatalf("unexpected error (is 'go' installed?): %v", err)
	}
}

func TestVerifyEnvStep_DryRun(t *testing.T) {
	ctx := newTestContext(t.TempDir(), true)
	ctx.ProjectType = "go"
	step := &VerifyEnvStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CheckGitStatusStep ---

func TestCheckGitStatusStep_CleanRepo(t *testing.T) {
	dir := setupGitProject(t)
	writeFile(t, dir, "file.txt", "hello")
	gitRun(t, dir, "-C", dir, "add", ".")
	gitRun(t, dir, "-C", dir, "commit", "-m", "init")

	ctx := newTestContext(dir, false)
	step := &CheckGitStatusStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckGitStatusStep_DirtyRepo_YesFlag(t *testing.T) {
	dir := setupGitProject(t)
	writeFile(t, dir, "file.txt", "hello")
	gitRun(t, dir, "-C", dir, "add", ".")
	gitRun(t, dir, "-C", dir, "commit", "-m", "init")
	writeFile(t, dir, "dirty.txt", "uncommitted")

	ctx := newTestContext(dir, false)
	ctx.Yes = true // auto-accept
	step := &CheckGitStatusStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error with --yes: %v", err)
	}
}

// --- CleanStep ---

func TestCleanStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module test\n\ngo 1.22\n")

	ctx := newTestContext(dir, true)
	ctx.ProjectType = "go"
	p, _ := providers.ForType("go")
	ctx.Provider = p

	step := &CleanStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanStep_WithCustomCommand_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.Config.Commands.Clean = "rm -rf build/"

	step := &CleanStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- BuildStep ---

func TestBuildStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.ProjectType = "go"
	p, _ := providers.ForType("go")
	ctx.Provider = p

	step := &BuildStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildStep_CustomCommand_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.Config.Commands.Build = "make release"

	step := &BuildStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- TestStep ---

func TestTestStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.ProjectType = "go"
	p, _ := providers.ForType("go")
	ctx.Provider = p

	step := &TestStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- VerifyStep ---

func TestVerifyStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.ProjectType = "go"

	step := &VerifyStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GitTagStep ---

func TestGitTagStep_DryRun(t *testing.T) {
	ctx := newTestContext(t.TempDir(), true)
	ctx.TagName = "v1.0.0"
	step := &GitTagStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitTagStep_CreateAndPush(t *testing.T) {
	// Set up bare remote + local clone
	remote := t.TempDir()
	local := t.TempDir()

	cmd := exec.Command("git", "init", "--bare", remote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %s: %v", out, err)
	}
	cmd = exec.Command("git", "clone", remote, local)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %s: %v", out, err)
	}
	gitRun(t, local, "-C", local, "config", "user.email", "test@test.com")
	gitRun(t, local, "-C", local, "config", "user.name", "Test")
	writeFile(t, local, "file.txt", "hello")
	gitRun(t, local, "-C", local, "add", ".")
	gitRun(t, local, "-C", local, "commit", "-m", "init")
	gitRun(t, local, "-C", local, "push", "origin", "HEAD")

	ctx := newTestContext(local, false)
	ctx.TagName = "v5.0.0"
	ctx.Version = "5.0.0"
	ctx.Yes = true

	step := &GitTagStep{}
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tag exists on remote
	cmd = exec.Command("git", "ls-remote", "--tags", "origin")
	cmd.Dir = local
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "refs/tags/v5.0.0") {
		t.Errorf("expected tag v5.0.0 on remote, got: %s", string(out))
	}
}

func TestGitTagStep_AlreadyOnRemote(t *testing.T) {
	remote := t.TempDir()
	local := t.TempDir()

	cmd := exec.Command("git", "init", "--bare", remote)
	cmd.CombinedOutput()
	cmd = exec.Command("git", "clone", remote, local)
	cmd.CombinedOutput()
	gitRun(t, local, "-C", local, "config", "user.email", "test@test.com")
	gitRun(t, local, "-C", local, "config", "user.name", "Test")
	writeFile(t, local, "file.txt", "hello")
	gitRun(t, local, "-C", local, "add", ".")
	gitRun(t, local, "-C", local, "commit", "-m", "init")
	gitRun(t, local, "-C", local, "push", "origin", "HEAD")
	gitRun(t, local, "-C", local, "tag", "-a", "v1.0.0", "-m", "Release")
	gitRun(t, local, "-C", local, "push", "origin", "v1.0.0")

	ctx := newTestContext(local, false)
	ctx.TagName = "v1.0.0"
	ctx.Version = "1.0.0"

	step := &GitTagStep{}
	// Should succeed (skip existing tag)
	if err := step.Execute(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- PublishStep ---

func TestPublishStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.ProjectType = "go"
	p, _ := providers.ForType("go")
	ctx.Provider = p

	step := &PublishStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GitHubReleaseStep ---

func TestGitHubReleaseStep_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx := newTestContext(dir, true)
	ctx.TagName = "v1.0.0"
	p, _ := providers.ForType("go")
	ctx.Provider = p

	step := &GitHubReleaseStep{}
	if err := step.DryRun(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
