package steps

import (
	"runtime"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

type BuildStep struct{}

func (s *BuildStep) Name() string        { return "build" }
func (s *BuildStep) Description() string  { return "Build project" }
func (s *BuildStep) Destructive() bool    { return false }

func (s *BuildStep) Execute(ctx *pipeline.Context) error {
	if ctx.Config != nil && ctx.Config.Commands.Build != "" {
		ctx.UI.Info("Using custom build command: %s", ctx.Config.Commands.Build)
		if runtime.GOOS == "windows" {
			_, err := ctx.Runner.Run("cmd", "/c", ctx.Config.Commands.Build)
			return err
		}
		_, err := ctx.Runner.Run("sh", "-c", ctx.Config.Commands.Build)
		return err
	}
	return ctx.Provider.Build(ctx)
}

func (s *BuildStep) DryRun(ctx *pipeline.Context) error {
	if ctx.Config != nil && ctx.Config.Commands.Build != "" {
		ctx.UI.DryRunMsg("Would run custom build: %s", ctx.Config.Commands.Build)
	} else {
		ctx.UI.DryRunMsg("Would build using %s provider", ctx.ProjectType)
	}
	return nil
}
