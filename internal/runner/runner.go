package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aiperceivable/unirelease/internal/ui"
)

// Runner executes external commands with dry-run support.
type Runner struct {
	DryRun bool
	Dir    string
	UI     *ui.UI
}

// New creates a Runner.
func New(dir string, dryRun bool, u *ui.UI) *Runner {
	return &Runner{DryRun: dryRun, Dir: dir, UI: u}
}

// Run executes a command, printing it first.
// In dry-run mode, prints the command but does not execute it.
func (r *Runner) Run(name string, args ...string) (string, error) {
	return r.RunSensitive(nil, name, args...)
}

// RunSensitive executes a command, masking arguments at the specified indices in the output.
func (r *Runner) RunSensitive(maskIndices []int, name string, args ...string) (string, error) {
	maskMap := make(map[int]bool)
	for _, idx := range maskIndices {
		maskMap[idx] = true
	}

	displayArgs := make([]string, len(args))
	for i, arg := range args {
		if maskMap[i] {
			displayArgs[i] = "********"
		} else {
			displayArgs[i] = arg
		}
	}

	cmdStr := formatCommand(name, displayArgs)

	if r.DryRun {
		r.UI.DryRunMsg("Would run: %s", cmdStr)
		return "", nil
	}

	r.UI.Command(cmdStr)
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %s: %w", cmdStr, err)
	}
	return "", nil
}

// formatCommand formats a command and its arguments for display, quoting args with spaces.
func formatCommand(name string, args []string) string {
	parts := make([]string, len(args)+1)
	parts[0] = quoteIfNecessary(name)
	for i, arg := range args {
		parts[i+1] = quoteIfNecessary(arg)
	}
	return strings.Join(parts, " ")
}

func quoteIfNecessary(s string) string {
	if strings.Contains(s, " ") || s == "" {
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}
	return s
}

// RunSilent executes a command and captures output (no printing).
func (r *Runner) RunSilent(name string, args ...string) (string, error) {
	if r.DryRun {
		return "", nil
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CommandExists checks if an executable is in PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
