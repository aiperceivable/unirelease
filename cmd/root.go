package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/detector"
	"github.com/aiperceivable/unirelease/internal/pipeline"
	"github.com/aiperceivable/unirelease/internal/pipeline/steps"
	"github.com/aiperceivable/unirelease/internal/providers"
	"github.com/aiperceivable/unirelease/internal/runner"
	"github.com/aiperceivable/unirelease/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagStep      string
	flagSkip      []string
	flagYes       bool
	flagDryRun    bool
	flagVersion   string
	flagType      string
	flagListSteps bool
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// buildAllSteps returns the canonical pipeline step list.
func buildAllSteps() []pipeline.Step {
	return []pipeline.Step{
		&steps.DetectStep{},
		&steps.ReadVersionStep{},
		&steps.VerifyEnvStep{},
		&steps.CheckGitStatusStep{},
		&steps.CleanStep{},
		&steps.BuildStep{},
		&steps.TestStep{},
		&steps.VerifyStep{},
		&steps.GitTagStep{},
		&steps.GitHubReleaseStep{},
		&steps.PublishStep{},
	}
}

// validStepNames returns the list of valid step names derived from buildAllSteps.
func validStepNames() []string {
	allSteps := buildAllSteps()
	names := make([]string, len(allSteps))
	for i, s := range allSteps {
		names[i] = s.Name()
	}
	return names
}

// buildLongHelp generates the Long help text from the step list.
func buildLongHelp() string {
	var b strings.Builder
	b.WriteString("Auto-detects project type (Rust, Node, Bun, Python, Go) and runs a unified release pipeline.\n")
	b.WriteString("\nAvailable steps (in execution order):\n")
	for i, step := range buildAllSteps() {
		tag := ""
		if step.Destructive() {
			tag = " [destructive]"
		}
		fmt.Fprintf(&b, "  %2d. %-17s %s%s\n", i+1, step.Name(), step.Description(), tag)
	}
	b.WriteString("\nUse --list-steps for detailed descriptions of each step.")
	return b.String()
}

var rootCmd = &cobra.Command{
	Use:           "unirelease [path]",
	Short:         "Unified release pipeline for any project",
	Args:          cobra.MaximumNArgs(1),
	RunE:          runRelease,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	rootCmd.Long = buildLongHelp()
	rootCmd.Flags().StringVar(&flagStep, "step", "", "Run only a specific pipeline step")
	rootCmd.Flags().StringSliceVar(&flagSkip, "skip", nil, "Steps to skip (comma-separated, e.g. --skip publish,test)")
	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Non-interactive mode (skip confirmations)")
	rootCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview pipeline without executing")
	rootCmd.Flags().StringVarP(&flagVersion, "set-version", "V", "", "Override detected version (e.g. 1.2.3)")
	rootCmd.Flags().StringVar(&flagType, "type", "", "Override auto-detection (rust|node|bun|python|go)")
	rootCmd.Flags().BoolVar(&flagListSteps, "list-steps", false, "Show detailed descriptions of all pipeline steps")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

// resolveProjectDir resolves the project directory from args or cwd.
func resolveProjectDir(args []string) (string, error) {
	var dir string
	if len(args) > 0 {
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
		dir = absPath
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		dir = cwd
	}

	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("directory not found: %s", dir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", dir)
	}
	return dir, nil
}

// validateFlags validates all flag values before running the pipeline.
func validateFlags() error {
	names := validStepNames()
	nameSet := make(map[string]bool, len(names))
	for _, s := range names {
		nameSet[s] = true
	}

	if flagStep != "" && !nameSet[flagStep] {
		return fmt.Errorf("invalid step %q (valid: %s)", flagStep, strings.Join(names, ", "))
	}

	if flagType != "" {
		valid := false
		for _, t := range detector.ValidTypes() {
			if detector.ProjectType(flagType) == t {
				valid = true
				break
			}
		}
		if !valid {
			types := detector.ValidTypes()
			strs := make([]string, len(types))
			for i, t := range types {
				strs[i] = string(t)
			}
			return fmt.Errorf("invalid type %q (valid: %s)", flagType, strings.Join(strs, ", "))
		}
	}

	for _, s := range flagSkip {
		if !nameSet[s] {
			return fmt.Errorf("invalid skip step %q (valid: %s)", s, strings.Join(names, ", "))
		}
	}

	if flagVersion != "" && !semverRegex.MatchString(flagVersion) {
		return fmt.Errorf("invalid version %q (expected format: X.Y.Z, e.g. 1.2.3)", flagVersion)
	}

	return nil
}

func runRelease(cmd *cobra.Command, args []string) error {
	// List steps mode
	if flagListSteps {
		printStepDetails()
		return nil
	}

	// Validate flags
	if err := validateFlags(); err != nil {
		return err
	}

	// Resolve project directory
	projectDir, err := resolveProjectDir(args)
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load(projectDir)
	if err != nil {
		return err
	}

	// Merge CLI flags into config
	cfg.Merge(flagType, flagSkip)

	// Create UI
	u := ui.New()

	// Create runner
	r := runner.New(projectDir, flagDryRun, u)

	// Build pipeline context
	ctx := &pipeline.Context{
		ProjectDir:      projectDir,
		VersionOverride: flagVersion,
		Config:          cfg,
		DryRun:          flagDryRun,
		Yes:             flagYes,
		Step:            flagStep,
		Runner:          r,
		UI:              u,
		TypeOverride:    flagType,
	}

	// Build step list
	allSteps := buildAllSteps()

	// Wrap detect step to also resolve the provider after detection
	wrappedSteps := make([]pipeline.Step, len(allSteps))
	copy(wrappedSteps, allSteps)
	wrappedSteps[0] = &detectAndResolveStep{inner: allSteps[0].(*steps.DetectStep)}

	engine := pipeline.NewEngine(ctx, wrappedSteps)

	// Run pipeline
	return engine.Run()
}

// detectAndResolveStep wraps DetectStep to also resolve the provider.
type detectAndResolveStep struct {
	inner *steps.DetectStep
}

func (s *detectAndResolveStep) Name() string        { return s.inner.Name() }
func (s *detectAndResolveStep) Description() string  { return s.inner.Description() }
func (s *detectAndResolveStep) Help() string         { return s.inner.Help() }
func (s *detectAndResolveStep) Destructive() bool    { return s.inner.Destructive() }

func (s *detectAndResolveStep) Execute(ctx *pipeline.Context) error {
	if err := s.inner.Execute(ctx); err != nil {
		return err
	}
	return resolveProvider(ctx)
}

func (s *detectAndResolveStep) DryRun(ctx *pipeline.Context) error {
	if err := s.inner.DryRun(ctx); err != nil {
		return err
	}
	return resolveProvider(ctx)
}

func resolveProvider(ctx *pipeline.Context) error {
	provider, err := providers.ForType(ctx.ProjectType)
	if err != nil {
		return err
	}
	ctx.Provider = provider
	return nil
}

func printStepDetails() {
	infoList := pipeline.StepInfoList(buildAllSteps())

	fmt.Println("Pipeline Steps:")
	fmt.Println()
	for i, info := range infoList {
		marker := " "
		if info.Destructive {
			marker = "!"
		}
		fmt.Printf("  %2d. [%s] %-17s %s\n", i+1, marker, info.Name, info.Description)
		fmt.Printf("      %s\n\n", info.Help)
	}
	fmt.Println("Legend: [!] = destructive (prompts for confirmation, use --yes to skip)")
}
