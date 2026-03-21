package steps

import (
	"github.com/aipartnerup/unirelease/internal/pipeline"
)

type VerifyStep struct{}

func (s *VerifyStep) Name() string        { return "verify" }
func (s *VerifyStep) Description() string { return "Verify package" }
func (s *VerifyStep) Destructive() bool   { return false }

func (s *VerifyStep) Execute(ctx *pipeline.Context) error {
	err := ctx.Provider.Verify(ctx)
	if err != nil {
		return err
	}
	ctx.UI.StepDone("Package verified")
	return nil
}

func (s *VerifyStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would verify package for %s project", ctx.ProjectType)
	return nil
}
