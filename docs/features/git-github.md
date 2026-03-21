# UNI-003: Git + GitHub Operations

| Field            | Value                                              |
|------------------|----------------------------------------------------|
| **Feature ID**   | UNI-003                                            |
| **Phase**        | 1 - Core Pipeline                                  |
| **Priority**     | P0                                                 |
| **Effort**       | M (3-5 days)                                       |
| **Dependencies** | UNI-002                                            |
| **Packages**     | `internal/git/`, `internal/github/`                |

---

## 1. Purpose

Implement the shared git operations (status check, branch detection, tag creation/push) and GitHub operations (token resolution, release creation, asset upload) used by all language providers. These are the shared steps in the pipeline that are identical across all project types.

---

## 2. Files to Create

```
internal/
  git/
    git.go
    git_test.go
  github/
    client.go
    client_test.go
```

---

## 3. Implementation Detail

### 3.1 internal/git/git.go

All git functions use `exec.Command("git", ...)` with the project directory set as `cmd.Dir`.

**Status:**

```go
// Status checks if the working tree is clean.
// Returns clean=true if `git status --porcelain` produces no output.
// Returns the raw status output for display when dirty.
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
```

**CurrentBranch:**

```go
// CurrentBranch returns the current branch name.
// Uses `git rev-parse --abbrev-ref HEAD`.
func CurrentBranch(dir string) (string, error) {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git branch: %w", err)
    }
    return strings.TrimSpace(string(out)), nil
}
```

**TagExists (local):**

```go
// TagExists checks if a tag exists locally.
func TagExists(dir string, tag string) bool {
    cmd := exec.Command("git", "rev-parse", tag)
    cmd.Dir = dir
    return cmd.Run() == nil
}
```

**TagExistsOnRemote:**

```go
// TagExistsOnRemote checks if a tag exists on the remote.
// Uses `git ls-remote --tags origin` and checks for exact match.
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
```

**CreateTag:**

```go
// CreateTag creates an annotated git tag.
func CreateTag(dir string, tag string, message string) error {
    cmd := exec.Command("git", "tag", "-a", tag, "-m", message)
    cmd.Dir = dir
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git tag: %s: %w", strings.TrimSpace(string(out)), err)
    }
    return nil
}
```

**PushTag:**

```go
// PushTag pushes a specific tag to origin.
func PushTag(dir string, tag string) error {
    cmd := exec.Command("git", "push", "origin", tag)
    cmd.Dir = dir
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git push tag: %s: %w", strings.TrimSpace(string(out)), err)
    }
    return nil
}
```

**RemoteURL:**

```go
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
```

**ParseGitHubRepo:**

```go
// ParseGitHubRepo extracts "owner/repo" from a GitHub remote URL.
// Handles SSH: git@github.com:owner/repo.git
// Handles HTTPS: https://github.com/owner/repo.git
// Returns error if the URL is not a GitHub URL.
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
```

### 3.2 internal/github/client.go

```go
package github

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"

    gh "github.com/google/go-github/v60/github"
    "golang.org/x/oauth2"
)

type Client struct {
    Token string
    Repo  string // "owner/repo"
    gh    *gh.Client
}
```

**ResolveToken:**

```go
// ResolveToken attempts to find a GitHub token from multiple sources.
// Priority order:
// 1. GITHUB_TOKEN environment variable
// 2. GH_TOKEN environment variable (GitHub CLI convention)
// 3. `gh auth token` command output (if gh CLI installed)
// 4. `git config github.token`
// Returns (token, source description, error).
func ResolveToken() (string, string, error) {
    // 1. GITHUB_TOKEN env
    if t := os.Getenv("GITHUB_TOKEN"); t != "" {
        return t, "GITHUB_TOKEN environment variable", nil
    }
    // 2. GH_TOKEN env
    if t := os.Getenv("GH_TOKEN"); t != "" {
        return t, "GH_TOKEN environment variable", nil
    }
    // 3. gh auth token
    cmd := exec.Command("gh", "auth", "token")
    if out, err := cmd.CombinedOutput(); err == nil {
        t := strings.TrimSpace(string(out))
        if t != "" {
            return t, "gh CLI auth", nil
        }
    }
    // 4. git config
    cmd = exec.Command("git", "config", "github.token")
    if out, err := cmd.CombinedOutput(); err == nil {
        t := strings.TrimSpace(string(out))
        if t != "" {
            return t, "git config github.token", nil
        }
    }
    return "", "", fmt.Errorf("no GitHub token found; set GITHUB_TOKEN env var, run 'gh auth login', or set 'git config --global github.token TOKEN'")
}
```

**NewClient:**

```go
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

// owner and repoName parse the "owner/repo" string.
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
```

**ReleaseExists:**

```go
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
```

**CreateRelease:**

```go
// CreateRelease creates a GitHub Release.
// Returns the release ID for asset uploads.
func (c *Client) CreateRelease(tag string, title string, body string) (int64, error) {
    release := &gh.RepositoryRelease{
        TagName: gh.String(tag),
        Name:    gh.String(title),
        Body:    gh.String(body),
        Draft:   gh.Bool(false),
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
```

**UploadAsset:**

```go
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
```

---

## 4. Test Cases

### 4.1 Git Tests (git_test.go)

| Test Name                              | Setup                                         | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestStatus_CleanRepo                   | `git init`, commit a file                     | clean=true, output=""               |
| TestStatus_DirtyRepo                   | `git init`, add untracked file                | clean=false, output contains filename |
| TestCurrentBranch                      | `git init` (default branch)                   | "main" or "master"                  |
| TestTagExists_True                     | Create tag "v1.0.0"                           | true                                |
| TestTagExists_False                    | No tags                                       | false                               |
| TestCreateTag                          | `git init`, commit                            | Tag created, `git tag -l` shows it  |
| TestParseGitHubRepo_SSH               | "git@github.com:owner/repo.git"               | "owner/repo"                        |
| TestParseGitHubRepo_HTTPS             | "https://github.com/owner/repo.git"           | "owner/repo"                        |
| TestParseGitHubRepo_HTTPS_NoGit       | "https://github.com/owner/repo"               | "owner/repo"                        |
| TestParseGitHubRepo_NonGitHub         | "https://gitlab.com/owner/repo.git"           | Error: not a GitHub URL             |

### 4.2 GitHub Client Tests (client_test.go)

Use `httptest.Server` to mock the GitHub API.

| Test Name                              | Mock Response                                 | Expected Result                     |
|----------------------------------------|-----------------------------------------------|-------------------------------------|
| TestResolveToken_EnvVar                | Set GITHUB_TOKEN env                          | Returns token, source "GITHUB_TOKEN" |
| TestResolveToken_GHToken               | Set GH_TOKEN env                              | Returns token, source "GH_TOKEN"    |
| TestResolveToken_None                  | No env, no gh CLI, no git config              | Error with instructions             |
| TestReleaseExists_True                 | 200 response with release JSON                | true                                |
| TestReleaseExists_False                | 404 response                                  | false                               |
| TestCreateRelease_Success              | 201 response with release JSON                | Returns release ID                  |
| TestCreateRelease_Failure              | 422 response (tag already has release)        | Error                               |
| TestUploadAsset_Success                | 201 response                                  | nil error                           |

---

## 5. Acceptance Criteria

- [ ] `check_git_status` step detects uncommitted changes and reports them.
- [ ] `check_git_status` step shows the current branch name.
- [ ] `git_tag` step creates an annotated tag with message "Release <version>".
- [ ] `git_tag` step pushes the tag to origin.
- [ ] `git_tag` step skips if tag already exists on remote (no error).
- [ ] `git_tag` step pushes existing local tag if not on remote.
- [ ] `github_release` step resolves token from GITHUB_TOKEN env, then gh CLI, then git config.
- [ ] `github_release` step skips with helpful message if no token found (not fatal).
- [ ] `github_release` step creates a release with title "Release <version>".
- [ ] `github_release` step skips if release already exists for the tag.
- [ ] `github_release` step uploads binary assets when `Provider.BinaryAssets()` returns paths.
- [ ] GitHub repo is correctly parsed from both SSH and HTTPS remote URLs.
- [ ] All git commands use the project directory, not the current working directory.
