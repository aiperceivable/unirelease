package steps

import (
	"fmt"

	"github.com/aiperceivable/unirelease/internal/detector"
	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type DetectStep struct{}

func (s *DetectStep) Name() string        { return "detect" }
func (s *DetectStep) Description() string  { return "Detect project type" }
func (s *DetectStep) Destructive() bool    { return false }

func (s *DetectStep) Help() string {
	return "Auto-detect the project type by scanning manifest files (Cargo.toml, package.json, pyproject.toml, go.mod). " +
		"Can be overridden with --type flag or [type] in .unirelease.toml."
}

func (s *DetectStep) Execute(ctx *pipeline.Context) error {
	typeOverride := ctx.TypeOverride
	if ctx.Config != nil && ctx.Config.Type != "" && typeOverride == "" {
		typeOverride = ctx.Config.Type
	}

	result, err := detector.Detect(ctx.ProjectDir, typeOverride)
	if err != nil {
		return err
	}

	ctx.ProjectType = string(result.Type)
	if result.Manifest != "" {
		ctx.UI.StepDone(fmt.Sprintf("Detected %s (from %s)", result.Type, result.Manifest))
	} else {
		ctx.UI.StepDone(fmt.Sprintf("Detected %s (override)", result.Type))
	}
	return nil
}

func (s *DetectStep) DryRun(ctx *pipeline.Context) error {
	typeOverride := ctx.TypeOverride
	if ctx.Config != nil && ctx.Config.Type != "" && typeOverride == "" {
		typeOverride = ctx.Config.Type
	}

	result, err := detector.Detect(ctx.ProjectDir, typeOverride)
	if err != nil {
		return err
	}

	ctx.ProjectType = string(result.Type)
	ctx.UI.DryRunMsg("Detected %s (from %s)", result.Type, result.Manifest)
	return nil
}
