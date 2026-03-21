package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupGitRepo creates a temp git repo with an initial commit and a bare remote "origin".
// Returns (localDir, remoteDir).
func setupGitRepo(t *testing.T) (string, string) {
	t.Helper()

	remote := t.TempDir()
	local := t.TempDir()

	// Create bare remote
	run(t, remote, "git", "init", "--bare", remote)

	// Clone it
	run(t, local, "git", "clone", remote, local)

	// Configure user for commits
	run(t, local, "git", "-C", local, "config", "user.email", "test@test.com")
	run(t, local, "git", "-C", local, "config", "user.name", "Test")

	// Create initial commit
	dummyFile := filepath.Join(local, "README.md")
	os.WriteFile(dummyFile, []byte("# test"), 0644)
	run(t, local, "git", "-C", local, "add", ".")
	run(t, local, "git", "-C", local, "commit", "-m", "initial commit")
	run(t, local, "git", "-C", local, "push", "origin", "HEAD")

	return local, remote
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %s: %v", name, args, string(out), err)
	}
}

// --- Status ---

func TestStatus_CleanRepo(t *testing.T) {
	local, _ := setupGitRepo(t)

	clean, output, err := Status(local)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !clean {
		t.Errorf("expected clean, got dirty: %s", output)
	}
}

func TestStatus_DirtyRepo(t *testing.T) {
	local, _ := setupGitRepo(t)

	// Create untracked file
	os.WriteFile(filepath.Join(local, "dirty.txt"), []byte("dirty"), 0644)

	clean, _, err := Status(local)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clean {
		t.Error("expected dirty working tree")
	}
}

func TestStatus_ModifiedFile(t *testing.T) {
	local, _ := setupGitRepo(t)

	// Modify tracked file
	os.WriteFile(filepath.Join(local, "README.md"), []byte("modified"), 0644)

	clean, output, err := Status(local)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clean {
		t.Error("expected dirty working tree")
	}
	if output == "" {
		t.Error("expected non-empty output for modified file")
	}
}

// --- CurrentBranch ---

func TestCurrentBranch(t *testing.T) {
	local, _ := setupGitRepo(t)

	branch, err := CurrentBranch(local)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default branch could be main or master depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

// --- Tag operations ---

func TestTagExists_False(t *testing.T) {
	local, _ := setupGitRepo(t)

	if TagExists(local, "v1.0.0") {
		t.Error("expected tag to not exist")
	}
}

func TestCreateTag_AndTagExists(t *testing.T) {
	local, _ := setupGitRepo(t)

	err := CreateTag(local, "v1.0.0", "Release 1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !TagExists(local, "v1.0.0") {
		t.Error("expected tag to exist after creation")
	}
}

func TestCreateTag_DuplicateFails(t *testing.T) {
	local, _ := setupGitRepo(t)

	CreateTag(local, "v1.0.0", "Release 1.0.0")
	err := CreateTag(local, "v1.0.0", "Duplicate")
	if err == nil {
		t.Error("expected error for duplicate tag")
	}
}

func TestPushTag_AndTagExistsOnRemote(t *testing.T) {
	local, _ := setupGitRepo(t)

	// Tag should not exist on remote initially
	exists, err := TagExistsOnRemote(local, "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected tag to not exist on remote")
	}

	// Create and push tag
	CreateTag(local, "v2.0.0", "Release 2.0.0")
	err = PushTag(local, "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error pushing tag: %v", err)
	}

	// Now should exist on remote
	exists, err = TagExistsOnRemote(local, "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected tag to exist on remote after push")
	}
}

func TestTagExistsOnRemote_NotPushed(t *testing.T) {
	local, _ := setupGitRepo(t)

	CreateTag(local, "v3.0.0", "Release 3.0.0")
	// Don't push

	exists, err := TagExistsOnRemote(local, "v3.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected tag to NOT exist on remote (not pushed)")
	}
}

// --- RemoteURL ---

func TestRemoteURL(t *testing.T) {
	local, remote := setupGitRepo(t)

	url, err := RemoteURL(local)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != remote {
		t.Errorf("expected %q, got %q", remote, url)
	}
}

func TestRemoteURL_NoRemote(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "git", "init", dir)

	_, err := RemoteURL(dir)
	if err == nil {
		t.Error("expected error for repo with no remote")
	}
}
