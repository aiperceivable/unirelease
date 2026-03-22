package steps

import (
	"runtime"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type TestStep struct{}

func (s *TestStep) Name() string        { return "test" }
func (s *TestStep) Description() string  { return "Run tests" }
func (s *TestStep) Destructive() bool    { return false }

func (s *TestStep) Execute(ctx *pipeline.Context) error {
	if ctx.Config != nil && ctx.Config.Commands.Test != "" {
		ctx.UI.Info("Using custom test command: %s", ctx.Config.Commands.Test)
		if runtime.GOOS == "windows" {
			_, err := ctx.Runner.Run("cmd", "/c", ctx.Config.Commands.Test)
			return err
		}
		_, err := ctx.Runner.Run("sh", "-c", ctx.Config.Commands.Test)
		return err
	}
	return ctx.Provider.Test(ctx)
}

func (s *TestStep) DryRun(ctx *pipeline.Context) error {
	if ctx.Config != nil && ctx.Config.Commands.Test != "" {
		ctx.UI.DryRunMsg("Would run custom test: %s", ctx.Config.Commands.Test)
	} else {
		ctx.UI.DryRunMsg("Would run tests using %s provider", ctx.ProjectType)
	}
	return nil
}
