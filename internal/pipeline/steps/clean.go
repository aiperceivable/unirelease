package steps

import (
	"runtime"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type CleanStep struct{}

func (s *CleanStep) Name() string        { return "clean" }
func (s *CleanStep) Description() string  { return "Clean build artifacts" }
func (s *CleanStep) Destructive() bool    { return false }

func (s *CleanStep) Help() string {
	return "Remove previous build artifacts to ensure a fresh build. " +
		"Rust: cargo clean. Node/Bun: rm dist/. Python: rm dist/ build/ *.egg-info. Go: go clean. " +
		"Can be overridden with [commands].clean in .unirelease.toml."
}

func (s *CleanStep) Execute(ctx *pipeline.Context) error {
	if ctx.Config != nil && ctx.Config.Commands.Clean != "" {
		ctx.UI.Info("Using custom clean command: %s", ctx.Config.Commands.Clean)
		if runtime.GOOS == "windows" {
			_, err := ctx.Runner.Run("cmd", "/c", ctx.Config.Commands.Clean)
			return err
		}
		_, err := ctx.Runner.Run("sh", "-c", ctx.Config.Commands.Clean)
		return err
	}
	return ctx.Provider.Clean(ctx)
}

func (s *CleanStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would clean build artifacts for %s project", ctx.ProjectType)
	return nil
}
