package git

import "testing"

func TestParseGitHubRepo_SSH(t *testing.T) {
	repo, err := ParseGitHubRepo("git@github.com:owner/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", repo)
	}
}

func TestParseGitHubRepo_HTTPS(t *testing.T) {
	repo, err := ParseGitHubRepo("https://github.com/owner/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", repo)
	}
}

func TestParseGitHubRepo_HTTPS_NoGit(t *testing.T) {
	repo, err := ParseGitHubRepo("https://github.com/owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %q", repo)
	}
}

func TestParseGitHubRepo_NonGitHub(t *testing.T) {
	_, err := ParseGitHubRepo("https://gitlab.com/owner/repo.git")
	if err == nil {
		t.Fatal("expected error for non-GitHub URL")
	}
}
