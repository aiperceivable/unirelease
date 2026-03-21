package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aipartnerup/unirelease/internal/config"
	"github.com/aipartnerup/unirelease/internal/detector"
	"github.com/aipartnerup/unirelease/internal/pipeline"
	"github.com/aipartnerup/unirelease/internal/pipeline/steps"
	"github.com/aipartnerup/unirelease/internal/providers"
	"github.com/aipartnerup/unirelease/internal/runner"
	"github.com/aipartnerup/unirelease/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagStep    string
	flagYes     bool
	flagDryRun  bool
	flagVersion string
	flagType    string
)

// Valid pipeline step names.
var validSteps = []string{
	"detect", "read_version", "verify_env", "check_git_status",
	"clean", "build", "test", "verify", "git_tag", "github_release", "publish",
}

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

var rootCmd = &cobra.Command{
	Use:   "unirelease [path]",
	Short: "Unified release pipeline for any project",
	Long:  "Auto-detects project type (Rust, Node, Bun, Python, Go) and runs a unified release pipeline.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRelease,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Flags().StringVar(&flagStep, "step", "", "Run only a specific pipeline step")
	rootCmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Non-interactive mode (skip confirmations)")
	rootCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview pipeline without executing")
	rootCmd.Flags().StringVarP(&flagVersion, "version", "v", "", "Override detected version")
	rootCmd.Flags().StringVar(&flagType, "type", "", "Override auto-detection (rust|node|bun|python|go)")
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
	if flagStep != "" {
		valid := false
		for _, s := range validSteps {
			if flagStep == s {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid step %q (valid: %s)", flagStep, strings.Join(validSteps, ", "))
		}
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

	if flagVersion != "" && !semverRegex.MatchString(flagVersion) {
		return fmt.Errorf("invalid version %q (expected format: X.Y.Z, e.g. 1.2.3)", flagVersion)
	}

	return nil
}

func runRelease(cmd *cobra.Command, args []string) error {
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
	cfg.Merge(flagType)

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
	allSteps := []pipeline.Step{
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

	// The detect step sets ctx.Provider, but we need a resolve step.
	// We wrap the detect step to also resolve the provider.
	// Actually, the detect step sets ctx.ProjectType. We resolve provider after detect.
	// Let's use a custom step wrapper. Better: use a ProviderResolveStep after detect.
	// Simplest: override the detect step to also resolve provider.

	// Create engine with a provider-resolving wrapper
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
