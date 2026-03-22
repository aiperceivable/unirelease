package providers

import (
	"bufio"
	"strings"

	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/pipeline"
	"github.com/aiperceivable/unirelease/internal/runner"
	"github.com/aiperceivable/unirelease/internal/ui"
)

// mockRunner records commands without executing them.
type mockRunner struct {
	Commands []string
}

func (m *mockRunner) record(name string, args ...string) {
	cmd := name
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}
	m.Commands = append(m.Commands, cmd)
}

func newTestUI() *ui.UI {
	reader := bufio.NewReader(strings.NewReader(""))
	return ui.NewWithReader(reader, false)
}

// newMockContext creates a pipeline.Context with a dry-run runner
// that records commands without executing.
func newMockContext(projectDir string) (*pipeline.Context, *mockRunner) {
	u := newTestUI()
	mock := &mockRunner{}
	r := runner.New(projectDir, true, u) // dry-run mode — won't execute

	ctx := &pipeline.Context{
		ProjectDir: projectDir,
		Config:     config.Default(),
		DryRun:     true,
		Yes:        true,
		Runner:     r,
		UI:         u,
	}
	return ctx, mock
}
