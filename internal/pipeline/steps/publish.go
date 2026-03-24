package steps

import (
	"fmt"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

type PublishStep struct{}

func (s *PublishStep) Name() string        { return "publish" }
func (s *PublishStep) Description() string { return "Publish to registry" }
func (s *PublishStep) Destructive() bool   { return true }

func (s *PublishStep) Help() string {
	return "Publish the package to its language registry. " +
		"Rust: cargo publish (crates.io). Node: npm publish (npm). Python: twine upload (PyPI). " +
		"Go: no-op (uses git tags). Checks if version already exists before publishing. " +
		"Prompts for confirmation (use --yes to skip). Supports pre_publish/post_publish hooks. [DESTRUCTIVE]"
}

func (s *PublishStep) Execute(ctx *pipeline.Context) error {
	// Pre-check: warn if version already exists on registry
	exists, err := ctx.Provider.RegistryCheck(ctx)
	if err != nil {
		ctx.UI.Warn("Could not check registry: %v", err)
	}
	if exists {
		ctx.UI.Warn("Version %s already exists on %s", ctx.Version, ctx.Provider.PublishTarget())
		if !ctx.Yes {
			if !ctx.UI.Confirm("Publish anyway?") {
				ctx.UI.Info("Skipping publish")
				return nil
			}
		} else {
			ctx.UI.Info("Skipping publish (version already exists)")
			return nil
		}
	}

	err = ctx.Provider.Publish(ctx)
	if err != nil {
		return err
	}
	ctx.UI.StepDone(fmt.Sprintf("Published to %s", ctx.Provider.PublishTarget()))
	return nil
}

func (s *PublishStep) DryRun(ctx *pipeline.Context) error {
	ctx.UI.DryRunMsg("Would publish to %s", ctx.Provider.PublishTarget())
	return nil
}
