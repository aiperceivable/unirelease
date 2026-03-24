package steps

import (
	"fmt"

	"github.com/aiperceivable/unirelease/internal/detector"
	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type ReadVersionStep struct{}

func (s *ReadVersionStep) Name() string        { return "read_version" }
func (s *ReadVersionStep) Description() string  { return "Read version" }
func (s *ReadVersionStep) Destructive() bool    { return false }

func (s *ReadVersionStep) Help() string {
	return "Read the current version from the project manifest (e.g. Cargo.toml [package].version, " +
		"package.json version, pyproject.toml [project].version, go build tags). " +
		"Can be overridden with --set-version flag."
}

func (s *ReadVersionStep) Execute(ctx *pipeline.Context) error {
	version, err := detector.ReadVersion(
		ctx.ProjectDir,
		detector.ProjectType(ctx.ProjectType),
		ctx.VersionOverride,
	)
	if err != nil {
		return err
	}

	ctx.Version = version
	ctx.TagName = ctx.FormatTag()
	ctx.UI.StepDone(fmt.Sprintf("Version %s, Tag %s", ctx.Version, ctx.TagName))
	return nil
}

func (s *ReadVersionStep) DryRun(ctx *pipeline.Context) error {
	return s.Execute(ctx) // Version reading is side-effect free
}
