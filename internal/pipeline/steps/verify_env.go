package steps

import (
	"fmt"
	"strings"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type VerifyEnvStep struct{}

func (s *VerifyEnvStep) Name() string        { return "verify_env" }
func (s *VerifyEnvStep) Description() string  { return "Verify environment" }
func (s *VerifyEnvStep) Destructive() bool    { return false }

func (s *VerifyEnvStep) Execute(ctx *pipeline.Context) error {
	if ctx.Provider == nil {
		return fmt.Errorf("no provider set (detect step must run first)")
	}

	missing, err := ctx.Provider.VerifyEnv()
	if err != nil {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	ctx.UI.StepDone("All required tools available")
	return nil
}

func (s *VerifyEnvStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would verify: required tools for %s", ctx.ProjectType)
	return nil
}
