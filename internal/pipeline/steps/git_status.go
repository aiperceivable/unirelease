package steps

import (
	"fmt"

	"github.com/aipartnerup/unirelease/internal/git"
	"github.com/aipartnerup/unirelease/internal/pipeline"
)

type CheckGitStatusStep struct{}

func (s *CheckGitStatusStep) Name() string        { return "check_git_status" }
func (s *CheckGitStatusStep) Description() string  { return "Check git status" }
func (s *CheckGitStatusStep) Destructive() bool    { return false }

func (s *CheckGitStatusStep) Execute(ctx *pipeline.Context) error {
	clean, output, err := git.Status(ctx.ProjectDir)
	if err != nil {
		return err
	}

	if !clean {
		ctx.UI.Warn("Working tree has uncommitted changes:\n%s", output)
		if !ctx.Yes {
			if !ctx.UI.Confirm("Continue with uncommitted changes?") {
				return fmt.Errorf("aborted: uncommitted changes")
			}
		}
	}

	branch, err := git.CurrentBranch(ctx.ProjectDir)
	if err != nil {
		return err
	}

	if clean {
		ctx.UI.StepDone(fmt.Sprintf("Clean working tree, branch: %s", branch))
	} else {
		ctx.UI.StepDone(fmt.Sprintf("Branch: %s (with uncommitted changes)", branch))
	}
	return nil
}

func (s *CheckGitStatusStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would check git status and branch")
	return nil
}
