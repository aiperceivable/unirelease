package pipeline

// Step represents a single pipeline step.
type Step interface {
	Name() string
	Description() string
	Help() string
	Execute(ctx *Context) error
	DryRun(ctx *Context) error
	Destructive() bool
}

// StepInfo holds displayable metadata for a pipeline step.
type StepInfo struct {
	Name        string
	Description string
	Help        string
	Destructive bool
}

// StepInfoList returns metadata for all registered steps.
func StepInfoList(steps []Step) []StepInfo {
	out := make([]StepInfo, len(steps))
	for i, s := range steps {
		out[i] = StepInfo{
			Name:        s.Name(),
			Description: s.Description(),
			Help:        s.Help(),
			Destructive: s.Destructive(),
		}
	}
	return out
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
	"verify",
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
