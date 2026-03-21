package steps

import (
	"fmt"

	"github.com/aipartnerup/unirelease/internal/git"
	"github.com/aipartnerup/unirelease/internal/pipeline"
)

type GitTagStep struct{}

func (s *GitTagStep) Name() string        { return "git_tag" }
func (s *GitTagStep) Description() string  { return "Create git tag" }
func (s *GitTagStep) Destructive() bool    { return true }

func (s *GitTagStep) Execute(ctx *pipeline.Context) error {
	tag := ctx.TagName

	// Check remote
	onRemote, err := git.TagExistsOnRemote(ctx.ProjectDir, tag)
	if err != nil {
		ctx.UI.Warn("Could not check remote tags: %v", err)
	}
	if onRemote {
		ctx.UI.StepDone(fmt.Sprintf("Tag %s already exists on remote", tag))
		return nil
	}

	// Check local
	if git.TagExists(ctx.ProjectDir, tag) {
		ctx.UI.Info("Tag %s exists locally, pushing to remote...", tag)
		if err := git.PushTag(ctx.ProjectDir, tag); err != nil {
			return err
		}
		ctx.UI.StepDone(fmt.Sprintf("Tag %s pushed to remote", tag))
		return nil
	}

	// Create and push
	if err := git.CreateTag(ctx.ProjectDir, tag, "Release "+ctx.Version); err != nil {
		return err
	}
	if err := git.PushTag(ctx.ProjectDir, tag); err != nil {
		return err
	}
	ctx.UI.StepDone(fmt.Sprintf("Tag %s created and pushed", tag))
	return nil
}

func (s *GitTagStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would create and push tag %s", ctx.TagName)
	return nil
}
