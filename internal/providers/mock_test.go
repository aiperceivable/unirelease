package providers

import (
	"bufio"
	"strings"

	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/pipeline"
	"github.com/aiperceivable/unirelease/internal/ui"
)

// mockRunner records commands and their sensitive/masked indices.
type mockRunner struct {
	Commands []string
	Masks    [][]int
}

func (m *mockRunner) Run(name string, args ...string) (string, error) {
	m.Commands = append(m.Commands, name+" "+strings.Join(args, " "))
	m.Masks = append(m.Masks, nil)
	return "", nil
}

func (m *mockRunner) RunSensitive(maskIndices []int, name string, args ...string) (string, error) {
	m.Commands = append(m.Commands, name+" "+strings.Join(args, " "))
	m.Masks = append(m.Masks, maskIndices)
	return "", nil
}

func (m *mockRunner) RunSilent(name string, args ...string) (string, error) {
	return "", nil
}

func newTestUI() *ui.UI {
	reader := bufio.NewReader(strings.NewReader(""))
	return ui.NewWithReader(reader, false)
}

// newMockContext creates a pipeline.Context with a mock runner that records commands.
func newMockContext(projectDir string) (*pipeline.Context, *mockRunner) {
	u := newTestUI()
	mock := &mockRunner{}
	
	ctx := &pipeline.Context{
		ProjectDir: projectDir,
		Config:     config.Default(),
		DryRun:     true,
		Yes:        true,
		Runner:     mock,
		UI:         u,
	}
	return ctx, mock
}
