package steps

import (
	"fmt"

	"github.com/aiperceivable/unirelease/internal/changelog"
	"github.com/aiperceivable/unirelease/internal/git"
	gh "github.com/aiperceivable/unirelease/internal/github"
	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type GitHubReleaseStep struct{}

func (s *GitHubReleaseStep) Name() string        { return "github_release" }
func (s *GitHubReleaseStep) Description() string  { return "Create GitHub Release" }
func (s *GitHubReleaseStep) Destructive() bool    { return true }

func (s *GitHubReleaseStep) Execute(ctx *pipeline.Context) error {
	// Resolve GitHub repo
	remoteURL, err := git.RemoteURL(ctx.ProjectDir)
	if err != nil {
		ctx.UI.Warn("Could not get git remote URL: %v", err)
		ctx.UI.Info("Skipping GitHub Release (no remote)")
		return nil
	}

	repo, err := git.ParseGitHubRepo(remoteURL)
	if err != nil {
		ctx.UI.Warn("Not a GitHub repo: %v", err)
		ctx.UI.Info("Skipping GitHub Release")
		return nil
	}
	ctx.GitHubRepo = repo

	// Resolve token
	token, source, err := gh.ResolveToken()
	if err != nil {
		ctx.UI.Warn("No GitHub token found")
		ctx.UI.Info("Set GITHUB_TOKEN env var, run 'gh auth login', or set 'git config --global github.token TOKEN'")
		ctx.UI.Info("Skipping GitHub Release")
		return nil
	}
	ctx.UI.Info("Using token from %s", source)

	// Create client
	client := gh.NewClient(token, repo)

	// Check if release exists
	exists, err := client.ReleaseExists(ctx.TagName)
	if err != nil {
		ctx.UI.Warn("Could not check release: %v", err)
	}
	if exists {
		ctx.UI.StepDone(fmt.Sprintf("Release %s already exists", ctx.TagName))
		return nil
	}

	// Extract release notes from CHANGELOG.md
	body := changelog.FormatReleaseBody(ctx.ProjectDir, ctx.Version)

	// Create release
	releaseID, err := client.CreateRelease(
		ctx.TagName,
		"Release "+ctx.Version,
		body,
	)
	if err != nil {
		return err
	}
	ctx.UI.StepDone(fmt.Sprintf("Release %s created", ctx.TagName))

	// Upload binary assets if provider has them
	if ctx.Provider != nil {
		assets, err := ctx.Provider.BinaryAssets(ctx)
		if err != nil {
			ctx.UI.Warn("Could not get binary assets: %v", err)
		}
		for _, asset := range assets {
			ctx.UI.Info("Uploading asset: %s", asset)
			if err := client.UploadAsset(releaseID, asset); err != nil {
				return fmt.Errorf("upload asset: %w", err)
			}
		}
	}

	return nil
}

func (s *GitHubReleaseStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would create GitHub Release for tag %s", ctx.TagName)
	if ctx.Provider != nil {
		assets, _ := ctx.Provider.BinaryAssets(ctx)
		for _, asset := range assets {
			ctx.UI.DryRunMsg("Would upload asset: %s", asset)
		}
	}
	return nil
}
