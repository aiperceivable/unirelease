package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Status checks if the working tree is clean.
func Status(dir string) (clean bool, output string, err error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", fmt.Errorf("git status: %w", err)
	}
	trimmed := strings.TrimSpace(string(out))
	return trimmed == "", trimmed, nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// TagExists checks if a tag exists locally.
func TagExists(dir string, tag string) bool {
	cmd := exec.Command("git", "rev-parse", tag)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// TagExistsOnRemote checks if a tag exists on the remote.
func TagExistsOnRemote(dir string, tag string) (bool, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "origin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git ls-remote: %w", err)
	}
	target := "refs/tags/" + tag
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), target) {
			return true, nil
		}
	}
	return false, nil
}

// CreateTag creates an annotated git tag.
func CreateTag(dir string, tag string, message string) error {
	cmd := exec.Command("git", "tag", "-a", tag, "-m", message)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git tag: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// PushTag pushes a specific tag to origin.
func PushTag(dir string, tag string) error {
	cmd := exec.Command("git", "push", "origin", tag)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push tag: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// RemoteURL returns the origin remote URL.
func RemoteURL(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git remote: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseGitHubRepo extracts "owner/repo" from a GitHub remote URL.
func ParseGitHubRepo(remoteURL string) (string, error) {
	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repo := strings.TrimPrefix(remoteURL, "git@github.com:")
		repo = strings.TrimSuffix(repo, ".git")
		return repo, nil
	}
	// HTTPS format: https://github.com/owner/repo.git
	if strings.Contains(remoteURL, "github.com/") {
		idx := strings.Index(remoteURL, "github.com/")
		repo := remoteURL[idx+len("github.com/"):]
		repo = strings.TrimSuffix(repo, ".git")
		return repo, nil
	}
	return "", fmt.Errorf("not a GitHub URL: %s", remoteURL)
}
