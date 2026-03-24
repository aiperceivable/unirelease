package pipeline

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/aiperceivable/unirelease/internal/git"
	gh "github.com/aiperceivable/unirelease/internal/github"
	"github.com/aiperceivable/unirelease/internal/ui"
)

// Engine orchestrates pipeline execution.
type Engine struct {
	steps []Step
	ctx   *Context
}

// NewEngine creates an engine with the given steps.
func NewEngine(ctx *Context, steps []Step) *Engine {
	return &Engine{
		steps: steps,
		ctx:   ctx,
	}
}

// Run executes the pipeline.
func (e *Engine) Run() error {
	// Single step mode
	if e.ctx.Step != "" {
		for i, step := range e.steps {
			if step.Name() == e.ctx.Step {
				return e.executeStep(i, step)
			}
		}
		return fmt.Errorf("unknown step: %q", e.ctx.Step)
	}

	// Full pipeline
	results := make([]ui.StepResult, 0, len(e.steps))

	for i, step := range e.steps {
		// Check skip from config
		if e.ctx.Config != nil && e.ctx.Config.HasSkip(step.Name()) {
			e.ctx.UI.StepSkip(step.Name(), "skipped")
			results = append(results, ui.StepResult{
				Name:        step.Name(),
				Description: step.Description(),
				Status:      ui.StepStatusSkipped,
			})
			continue
		}

		// Run pre-hooks
		if err := e.runPreHook(step.Name()); err != nil {
			return fmt.Errorf("pre-%s hook: %w", step.Name(), err)
		}

		// Execute step
		err := e.executeStep(i, step)
		if err != nil {
			if errors.Is(err, ErrNoPublish) {
				e.ctx.UI.StepSkip(step.Name(), "not applicable")
				results = append(results, ui.StepResult{
					Name:        step.Name(),
					Description: step.Description(),
					Status:      ui.StepStatusSkipped,
				})
			} else if errors.Is(err, errUserDeclined) {
				e.ctx.UI.StepSkip(step.Name(), "user declined")
				results = append(results, ui.StepResult{
					Name:        step.Name(),
					Description: step.Description(),
					Status:      ui.StepStatusSkipped,
				})
			} else {
				results = append(results, ui.StepResult{
					Name:        step.Name(),
					Description: step.Description(),
					Status:      ui.StepStatusFailed,
				})
				return fmt.Errorf("step %s: %w", step.Name(), err)
			}
		} else {
			status := ui.StepStatusDone
			if e.ctx.DryRun {
				status = ui.StepStatusDryRun
			}
			results = append(results, ui.StepResult{
				Name:        step.Name(),
				Description: step.Description(),
				Status:      status,
			})
		}

		// Run post-hooks
		if err := e.runPostHook(step.Name()); err != nil {
			return fmt.Errorf("post-%s hook: %w", step.Name(), err)
		}
	}

	// Collect remote status for summary (non-blocking, best effort)
	summary := ui.SummaryData{
		Version:     e.ctx.Version,
		Tag:         e.ctx.TagName,
		ProjectType: e.ctx.ProjectType,
		Steps:       results,
	}
	e.collectRemoteStatus(&summary)

	// Print summary
	e.ctx.UI.Summary(summary)

	return nil
}

var errUserDeclined = errors.New("user declined")

func (e *Engine) executeStep(index int, step Step) error {
	e.ctx.UI.StepHeader(index+1, len(e.steps), step.Description())

	if e.ctx.DryRun {
		return step.DryRun(e.ctx)
	}

	// Destructive step prompt
	if step.Destructive() && !e.ctx.Yes {
		if !e.ctx.UI.Confirm(fmt.Sprintf("About to %s. Continue?", step.Description())) {
			return errUserDeclined
		}
	}

	return step.Execute(e.ctx)
}

func (e *Engine) runPreHook(stepName string) error {
	if e.ctx.Config == nil {
		return nil
	}
	var hookCmd string
	switch stepName {
	case "build":
		hookCmd = e.ctx.Config.Hooks.PreBuild
	case "publish":
		hookCmd = e.ctx.Config.Hooks.PrePublish
	}
	return e.executeHook(hookCmd, "pre_"+stepName)
}

func (e *Engine) runPostHook(stepName string) error {
	if e.ctx.Config == nil {
		return nil
	}
	var hookCmd string
	switch stepName {
	case "build":
		hookCmd = e.ctx.Config.Hooks.PostBuild
	case "publish":
		hookCmd = e.ctx.Config.Hooks.PostPublish
	}
	return e.executeHook(hookCmd, "post_"+stepName)
}

// collectRemoteStatus checks remote systems for the summary display.
// All checks are best-effort — failures are silently ignored.
func (e *Engine) collectRemoteStatus(summary *ui.SummaryData) {
	if e.ctx.DryRun || e.ctx.TagName == "" {
		return
	}

	// Check git tag on remote
	if onRemote, err := git.TagExistsOnRemote(e.ctx.ProjectDir, e.ctx.TagName); err == nil {
		summary.TagOnRemote = &onRemote
	}

	// Check GitHub Release
	remoteURL, err := git.RemoteURL(e.ctx.ProjectDir)
	if err == nil {
		if repo, err := git.ParseGitHubRepo(remoteURL); err == nil {
			if token, _, err := gh.ResolveToken(); err == nil {
				client := gh.NewClient(token, repo)
				if exists, err := client.ReleaseExists(e.ctx.TagName); err == nil {
					summary.ReleaseExists = &exists
				}
			}
		}
	}

	// Check registry
	if e.ctx.Provider != nil {
		target := e.ctx.Provider.PublishTarget()
		if target != "" && target != "GitHub Release" {
			summary.RegistryName = target
			if exists, err := e.ctx.Provider.RegistryCheck(e.ctx); err == nil {
				summary.RegistryExists = &exists
			}
		}
	}
}

func (e *Engine) executeHook(hookCmd string, hookName string) error {
	if hookCmd == "" {
		return nil
	}
	e.ctx.UI.Info("Running %s hook: %s", hookName, hookCmd)
	if runtime.GOOS == "windows" {
		_, err := e.ctx.Runner.Run("cmd", "/c", hookCmd)
		return err
	}
	_, err := e.ctx.Runner.Run("sh", "-c", hookCmd)
	return err
}
