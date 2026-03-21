package cmd

import (
	"os"
	"path/filepath"
	"testing"
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
