package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// StepStatus represents the outcome of a pipeline step.
type StepStatus int

const (
	StepStatusDone    StepStatus = iota
	StepStatusSkipped
	StepStatusFailed
	StepStatusDryRun
)

// StepResult holds the result of a single pipeline step.
type StepResult struct {
	Name        string
	Description string
	Status      StepStatus
}

// SummaryData holds data for the final release summary.
type SummaryData struct {
	Version        string
	Tag            string
	ProjectType    string
	Steps          []StepResult
	PublishURL     string
	ReleaseURL     string
	TagOnRemote    *bool // nil = not checked, true/false = result
	ReleaseExists  *bool
	RegistryExists *bool
	RegistryName   string
}

var (
	colorRed    = color.New(color.FgRed)
	colorGreen  = color.New(color.FgGreen)
	colorYellow = color.New(color.FgYellow)
	colorBlue   = color.New(color.FgBlue)
	colorCyan   = color.New(color.FgCyan)
)

// UI handles all user-facing output and input.
type UI struct {
	IsTTY  bool
	reader *bufio.Reader
}

// New creates a UI instance, detecting TTY status.
func New() *UI {
	return &UI{
		IsTTY:  term.IsTerminal(int(os.Stdout.Fd())),
		reader: bufio.NewReader(os.Stdin),
	}
}

// NewWithReader creates a UI with a custom reader (for testing).
func NewWithReader(reader *bufio.Reader, isTTY bool) *UI {
	return &UI{
		IsTTY:  isTTY,
		reader: reader,
	}
}

// Header prints a boxed header.
func (u *UI) Header(projectType string, version string, tag string) {
	colorCyan.Println("╔══════════════════════════════════════════════════════════╗")
	colorCyan.Printf("║  unirelease - %s release v%s\n", projectType, version)
	colorCyan.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Printf("  Type:    %s\n", colorCyan.Sprint(projectType))
	fmt.Printf("  Version: %s\n", colorCyan.Sprint(version))
	fmt.Printf("  Tag:     %s\n", colorCyan.Sprint(tag))
	fmt.Println()
}

// StepHeader prints the step progress line.
func (u *UI) StepHeader(current int, total int, description string) {
	colorBlue.Printf("[%d/%d] %s...\n", current, total, description)
}

// StepSkip prints a skip message.
func (u *UI) StepSkip(stepName string, reason string) {
	colorYellow.Printf("[skip] %s (%s)\n", stepName, reason)
}

// StepDone prints a success message.
func (u *UI) StepDone(message string) {
	colorGreen.Printf("  done: %s\n", message)
}

// Info prints an informational message.
func (u *UI) Info(format string, args ...interface{}) {
	colorCyan.Printf("  "+format+"\n", args...)
}

// Warn prints a warning message.
func (u *UI) Warn(format string, args ...interface{}) {
	colorYellow.Printf("  warning: "+format+"\n", args...)
}

// Error prints an error message.
func (u *UI) Error(format string, args ...interface{}) {
	colorRed.Printf("  error: "+format+"\n", args...)
}

// Command prints the command being executed.
func (u *UI) Command(cmdStr string) {
	fmt.Printf("  $ %s\n", cmdStr)
}

// DryRunMsg prints a dry-run preview message.
func (u *UI) DryRunMsg(format string, args ...interface{}) {
	colorYellow.Printf("  [dry-run] "+format+"\n", args...)
}

// Confirm displays a yes/no prompt. Default is yes.
// Non-TTY environments return false with a warning.
func (u *UI) Confirm(message string) bool {
	if !u.IsTTY {
		colorYellow.Printf("  %s [Y/n] (non-interactive, defaulting to no; use --yes)\n", message)
		return false
	}

	colorYellow.Printf("  %s [Y/n] ", message)
	input, err := u.reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "" || input == "y" || input == "yes"
}

// Summary prints the final release summary.
func (u *UI) Summary(results SummaryData) {
	fmt.Println()
	colorCyan.Println("╔══════════════════════════════════════════════════════════╗")
	colorCyan.Println("║  Release Summary                                        ║")
	colorCyan.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Version:  %s\n", colorCyan.Sprint(results.Version))
	fmt.Printf("  Tag:      %s\n", colorCyan.Sprint(results.Tag))
	fmt.Printf("  Type:     %s\n", colorCyan.Sprint(results.ProjectType))
	fmt.Println()

	for _, step := range results.Steps {
		switch step.Status {
		case StepStatusDone:
			colorGreen.Printf("  [done]    %s\n", step.Description)
		case StepStatusDryRun:
			colorYellow.Printf("  [dry-run] %s\n", step.Description)
		case StepStatusSkipped:
			colorYellow.Printf("  [skip]    %s\n", step.Description)
		case StepStatusFailed:
			colorRed.Printf("  [fail]  %s\n", step.Description)
		}
	}

	// Remote status checks
	if results.TagOnRemote != nil || results.ReleaseExists != nil || results.RegistryExists != nil {
		fmt.Println()
		colorCyan.Println("  Status:")
		if results.TagOnRemote != nil {
			fmt.Printf("    Git Tag:        %s\n", statusIcon(*results.TagOnRemote))
		}
		if results.ReleaseExists != nil {
			fmt.Printf("    GitHub Release: %s\n", statusIcon(*results.ReleaseExists))
		}
		if results.RegistryExists != nil && results.RegistryName != "" {
			fmt.Printf("    %-15s %s\n", results.RegistryName+":", statusIcon(*results.RegistryExists))
		}
	}

	if results.PublishURL != "" {
		fmt.Println()
		fmt.Printf("  Published: %s\n", colorCyan.Sprint(results.PublishURL))
	}
	if results.ReleaseURL != "" {
		fmt.Printf("  Release:   %s\n", colorCyan.Sprint(results.ReleaseURL))
	}

	fmt.Println()
	colorGreen.Println("  Release complete!")
	fmt.Println()
}

func statusIcon(ok bool) string {
	if ok {
		return colorGreen.Sprint("yes")
	}
	return colorRed.Sprint("no")
}
