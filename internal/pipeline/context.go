package pipeline

import (
	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/ui"
)

// Provider defines the contract for language-specific release operations.
// Declared here to avoid import cycles with providers package.
type Provider interface {
	Name() string
	VerifyEnv() ([]string, error)
	Clean(ctx *Context) error
	Build(ctx *Context) error
	Test(ctx *Context) error
	Verify(ctx *Context) error
	Publish(ctx *Context) error
	PublishTarget() string
	BinaryAssets(ctx *Context) ([]string, error)
	RegistryCheck(ctx *Context) (exists bool, err error)
}

// Runner defines the contract for executing external commands.
type Runner interface {
	Run(name string, args ...string) (string, error)
	RunSensitive(maskIndices []int, name string, args ...string) (string, error)
	RunSilent(name string, args ...string) (string, error)
}

// Context carries shared state through the pipeline.
type Context struct {
	ProjectDir      string
	ProjectType     string
	Version         string
	VersionOverride string
	TagName         string
	Provider        Provider
	Config          *config.Config
	DryRun          bool
	Yes             bool
	Step            string // if non-empty, run only this step
	Runner          Runner
	UI              *ui.UI
	GitHubRepo      string
	GitHubToken     string
	TypeOverride    string
	OTP             string   // one-time password for registry publish
	PublishArgs     []string // additional arguments for the publish command
}

// FormatTag applies the tag prefix to the version.
func (ctx *Context) FormatTag() string {
	prefix := "v"
	if ctx.Config != nil && ctx.Config.TagPrefix != "" {
		prefix = ctx.Config.TagPrefix
	}
	return prefix + ctx.Version
}
