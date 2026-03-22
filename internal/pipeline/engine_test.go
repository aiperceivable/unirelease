package pipeline

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/aiperceivable/unirelease/internal/config"
	"github.com/aiperceivable/unirelease/internal/ui"
)

// mockStep implements Step for testing.
type mockStep struct {
	name        string
	desc        string
	destructive bool
	execErr     error
	dryRunErr   error
	executed    bool
	dryRan      bool
}

func (s *mockStep) Name() string        { return s.name }
func (s *mockStep) Description() string  { return s.desc }
func (s *mockStep) Destructive() bool    { return s.destructive }

func (s *mockStep) Execute(ctx *Context) error {
	s.executed = true
	return s.execErr
}

func (s *mockStep) DryRun(ctx *Context) error {
	s.dryRan = true
	return s.dryRunErr
}

func newTestUI() *ui.UI {
	reader := bufio.NewReader(strings.NewReader(""))
	return ui.NewWithReader(reader, false)
}

func newTestContext() *Context {
	return &Context{
		Config:      config.Default(),
		UI:          newTestUI(),
		Version:     "1.0.0",
		TagName:     "v1.0.0",
		ProjectType: "rust",
	}
}

func TestEngine_FullPipeline(t *testing.T) {
	s1 := &mockStep{name: "step1", desc: "Step 1"}
	s2 := &mockStep{name: "step2", desc: "Step 2"}
	ctx := newTestContext()
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1, s2})
	err := engine.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s1.executed || !s2.executed {
		t.Error("expected both steps to be executed")
	}
}

func TestEngine_SingleStep(t *testing.T) {
	s1 := &mockStep{name: "build", desc: "Build"}
	s2 := &mockStep{name: "test", desc: "Test"}
	ctx := newTestContext()
	ctx.Step = "build"
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1, s2})
	err := engine.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s1.executed {
		t.Error("expected build step to be executed")
	}
	if s2.executed {
		t.Error("expected test step NOT to be executed")
	}
}

func TestEngine_SingleStep_Invalid(t *testing.T) {
	s1 := &mockStep{name: "build", desc: "Build"}
	ctx := newTestContext()
	ctx.Step = "bogus"

	engine := NewEngine(ctx, []Step{s1})
	err := engine.Run()
	if err == nil {
		t.Fatal("expected error for invalid step")
	}
}

func TestEngine_SkipConfig(t *testing.T) {
	s1 := &mockStep{name: "test", desc: "Test"}
	s2 := &mockStep{name: "build", desc: "Build"}
	ctx := newTestContext()
	ctx.Config.Skip = []string{"test"}
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1, s2})
	err := engine.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.executed {
		t.Error("expected test step to be skipped")
	}
	if !s2.executed {
		t.Error("expected build step to be executed")
	}
}

func TestEngine_DryRun(t *testing.T) {
	s1 := &mockStep{name: "build", desc: "Build"}
	ctx := newTestContext()
	ctx.DryRun = true

	engine := NewEngine(ctx, []Step{s1})
	err := engine.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1.executed {
		t.Error("expected step NOT to be executed in dry-run")
	}
	if !s1.dryRan {
		t.Error("expected DryRun to be called")
	}
}

func TestEngine_ErrNoPublish_SkipsStep(t *testing.T) {
	s1 := &mockStep{name: "publish", desc: "Publish", execErr: ErrNoPublish}
	ctx := newTestContext()
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1})
	err := engine.Run()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEngine_StepError_StopsPipeline(t *testing.T) {
	s1 := &mockStep{name: "build", desc: "Build", execErr: errors.New("build failed")}
	s2 := &mockStep{name: "test", desc: "Test"}
	ctx := newTestContext()
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1, s2})
	err := engine.Run()
	if err == nil {
		t.Fatal("expected error from failed step")
	}
	if s2.executed {
		t.Error("expected test step NOT to execute after build failure")
	}
}

func TestEngine_DestructiveNoPrompt_YesFlag(t *testing.T) {
	s1 := &mockStep{name: "publish", desc: "Publish", destructive: true}
	ctx := newTestContext()
	ctx.Yes = true

	engine := NewEngine(ctx, []Step{s1})
	err := engine.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s1.executed {
		t.Error("expected step to execute with --yes")
	}
}
