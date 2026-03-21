package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gh "github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API.
type Client struct {
	Token string
	Repo  string // "owner/repo"
	gh    *gh.Client
}

// ResolveToken attempts to find a GitHub token from multiple sources.
// Priority: GITHUB_TOKEN env > GH_TOKEN env > gh auth token > git config.
func ResolveToken() (string, string, error) {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, "GITHUB_TOKEN environment variable", nil
	}
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t, "GH_TOKEN environment variable", nil
	}
	cmd := exec.Command("gh", "auth", "token")
	if out, err := cmd.CombinedOutput(); err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, "gh CLI auth", nil
		}
	}
	cmd = exec.Command("git", "config", "github.token")
	if out, err := cmd.CombinedOutput(); err == nil {
		t := strings.TrimSpace(string(out))
		if t != "" {
			return t, "git config github.token", nil
		}
	}
	return "", "", fmt.Errorf("no GitHub token found; set GITHUB_TOKEN env var, run 'gh auth login', or set 'git config --global github.token TOKEN'")
}

// NewClient creates a GitHub API client.
func NewClient(token string, repo string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{
		Token: token,
		Repo:  repo,
		gh:    gh.NewClient(tc),
	}
}

// NewClientWithGH creates a Client with a pre-configured go-github client (for testing).
func NewClientWithGH(ghClient *gh.Client, repo string) *Client {
	return &Client{
		Token: "test-token",
		Repo:  repo,
		gh:    ghClient,
	}
}

func (c *Client) owner() string {
	parts := strings.SplitN(c.Repo, "/", 2)
	return parts[0]
}

func (c *Client) repoName() string {
	parts := strings.SplitN(c.Repo, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// ReleaseExists checks if a GitHub release exists for the given tag.
func (c *Client) ReleaseExists(tag string) (bool, error) {
	_, resp, err := c.gh.Repositories.GetReleaseByTag(
		context.Background(), c.owner(), c.repoName(), tag,
	)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("check release: %w", err)
	}
	return true, nil
}

// CreateRelease creates a GitHub Release. Returns the release ID.
func (c *Client) CreateRelease(tag string, title string, body string) (int64, error) {
	release := &gh.RepositoryRelease{
		TagName:    gh.String(tag),
		Name:       gh.String(title),
		Body:       gh.String(body),
		Draft:      gh.Bool(false),
		Prerelease: gh.Bool(false),
	}
	rel, _, err := c.gh.Repositories.CreateRelease(
		context.Background(), c.owner(), c.repoName(), release,
	)
	if err != nil {
		return 0, fmt.Errorf("create release: %w", err)
	}
	return rel.GetID(), nil
}

// UploadAsset uploads a binary file as a GitHub release asset.
func (c *Client) UploadAsset(releaseID int64, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open asset %s: %w", filePath, err)
	}
	defer f.Close()

	name := filepath.Base(filePath)
	opts := &gh.UploadOptions{Name: name}
	_, _, err = c.gh.Repositories.UploadReleaseAsset(
		context.Background(), c.owner(), c.repoName(), releaseID, opts, f,
	)
	if err != nil {
		return fmt.Errorf("upload asset %s: %w", name, err)
	}
	return nil
}
