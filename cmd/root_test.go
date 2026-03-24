package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

func TestValidateFlags_ValidStep(t *testing.T) {
	flagStep = "build"
	flagType = ""
	flagVersion = ""
	defer func() { flagStep = "" }()

	if err := validateFlags(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlags_InvalidStep(t *testing.T) {
	flagStep = "bogus"
	flagType = ""
	flagVersion = ""
	defer func() { flagStep = "" }()

	err := validateFlags()
	if err == nil {
		t.Fatal("expected error for invalid step")
	}
}

func TestValidateFlags_ValidType(t *testing.T) {
	flagStep = ""
	flagType = "rust"
	flagVersion = ""
	defer func() { flagType = "" }()

	if err := validateFlags(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlags_InvalidType(t *testing.T) {
	flagStep = ""
	flagType = "java"
	flagVersion = ""
	defer func() { flagType = "" }()

	err := validateFlags()
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestValidateFlags_ValidVersion(t *testing.T) {
	flagStep = ""
	flagType = ""
	flagVersion = "1.2.3"
	defer func() { flagVersion = "" }()

	if err := validateFlags(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlags_InvalidVersion(t *testing.T) {
	flagStep = ""
	flagType = ""
	flagVersion = "abc"
	defer func() { flagVersion = "" }()

	err := validateFlags()
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestValidateFlags_ValidSkip(t *testing.T) {
	flagStep = ""
	flagType = ""
	flagVersion = ""
	flagSkip = []string{"publish", "test"}
	defer func() { flagSkip = nil }()

	if err := validateFlags(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlags_InvalidSkip(t *testing.T) {
	flagStep = ""
	flagType = ""
	flagVersion = ""
	flagSkip = []string{"publish", "bogus"}
	defer func() { flagSkip = nil }()

	err := validateFlags()
	if err == nil {
		t.Fatal("expected error for invalid skip step")
	}
}

func TestBuildAllSteps_MatchesStepNames(t *testing.T) {
	allSteps := buildAllSteps()
	stepNames := pipeline.StepNames

	if len(allSteps) != len(stepNames) {
		t.Fatalf("buildAllSteps() has %d steps, StepNames has %d", len(allSteps), len(stepNames))
	}
	for i, step := range allSteps {
		if step.Name() != stepNames[i] {
			t.Errorf("step %d: buildAllSteps() has %q, StepNames has %q", i, step.Name(), stepNames[i])
		}
	}
}

func TestBuildAllSteps_HelpNotEmpty(t *testing.T) {
	for _, step := range buildAllSteps() {
		if step.Help() == "" {
			t.Errorf("step %q has empty Help()", step.Name())
		}
	}
}

func TestBuildLongHelp_ContainsAllSteps(t *testing.T) {
	helpText := buildLongHelp()
	for _, step := range buildAllSteps() {
		if !strings.Contains(helpText, step.Name()) {
			t.Errorf("Long help missing step %q", step.Name())
		}
	}
}

func TestResolveProjectDir_Explicit(t *testing.T) {
	dir := t.TempDir()
	result, err := resolveProjectDir([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != dir {
		t.Errorf("expected %s, got %s", dir, result)
	}
}

func TestResolveProjectDir_NonExistent(t *testing.T) {
	_, err := resolveProjectDir([]string{"/nonexistent/path/xyz"})
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestResolveProjectDir_UseCwd(t *testing.T) {
	cwd, _ := os.Getwd()
	result, err := resolveProjectDir([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != cwd {
		t.Errorf("expected cwd %s, got %s", cwd, result)
	}
}

func TestResolveProjectDir_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	os.WriteFile(file, []byte("test"), 0644)

	_, err := resolveProjectDir([]string{file})
	if err == nil {
		t.Fatal("expected error for file path")
	}
}
