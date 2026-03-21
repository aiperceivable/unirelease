package pipeline

// Step represents a single pipeline step.
type Step interface {
	Name() string
	Description() string
	Execute(ctx *Context) error
	DryRun(ctx *Context) error
	Destructive() bool
}

// StepNames defines the canonical order and valid step names.
var StepNames = []string{
	"detect",
	"read_version",
	"verify_env",
	"check_git_status",
	"clean",
	"build",
	"test",
	"git_tag",
	"github_release",
	"publish",
}

// ValidStepName checks if a name is a recognized pipeline step.
func ValidStepName(name string) bool {
	for _, n := range StepNames {
		if n == name {
			return true
		}
	}
	return false
}
