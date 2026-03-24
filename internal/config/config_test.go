package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TagPrefix != "v" {
		t.Errorf("expected default TagPrefix 'v', got %q", cfg.TagPrefix)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", "")
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TagPrefix != "v" {
		t.Errorf("expected default TagPrefix 'v', got %q", cfg.TagPrefix)
	}
}

func TestLoad_TypeOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", `type = "rust"`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Type != "rust" {
		t.Errorf("expected type 'rust', got %q", cfg.Type)
	}
}

func TestLoad_FullConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", `
type = "node"
tag_prefix = "release/v"
skip = ["clean", "test"]

[hooks]
pre_build = "make gen"
post_publish = "notify.sh"

[commands]
build = "make release"
`)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Type != "node" {
		t.Errorf("expected type 'node', got %q", cfg.Type)
	}
	if cfg.TagPrefix != "release/v" {
		t.Errorf("expected tag_prefix 'release/v', got %q", cfg.TagPrefix)
	}
	if len(cfg.Skip) != 2 {
		t.Errorf("expected 2 skip steps, got %d", len(cfg.Skip))
	}
	if cfg.Hooks.PreBuild != "make gen" {
		t.Errorf("expected pre_build 'make gen', got %q", cfg.Hooks.PreBuild)
	}
	if cfg.Commands.Build != "make release" {
		t.Errorf("expected build command 'make release', got %q", cfg.Commands.Build)
	}
}

func TestLoad_InvalidType(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", `type = "java"`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestLoad_InvalidSkipStep(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", `skip = ["bogus"]`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid skip step")
	}
}

func TestLoad_MalformedToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".unirelease.toml", `this is not valid {{{{`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for malformed TOML")
	}
}

func TestMerge_CLIOverridesConfig(t *testing.T) {
	cfg := &Config{Type: "rust"}
	cfg.Merge("node", nil)
	if cfg.Type != "node" {
		t.Errorf("expected 'node', got %q", cfg.Type)
	}
}

func TestMerge_CLIEmpty_KeepsConfig(t *testing.T) {
	cfg := &Config{Type: "rust"}
	cfg.Merge("", nil)
	if cfg.Type != "rust" {
		t.Errorf("expected 'rust', got %q", cfg.Type)
	}
}

func TestMerge_CLISkip(t *testing.T) {
	cfg := &Config{Skip: []string{"clean"}}
	cfg.Merge("", []string{"publish", "test"})
	if len(cfg.Skip) != 3 {
		t.Errorf("expected 3 skip steps, got %d: %v", len(cfg.Skip), cfg.Skip)
	}
	if !cfg.HasSkip("clean") || !cfg.HasSkip("publish") || !cfg.HasSkip("test") {
		t.Errorf("expected all three steps in skip list, got %v", cfg.Skip)
	}
}

func TestMerge_CLISkipDedup(t *testing.T) {
	cfg := &Config{Skip: []string{"clean", "test"}}
	cfg.Merge("", []string{"clean", "publish"})
	if len(cfg.Skip) != 3 {
		t.Errorf("expected 3 skip steps (deduped), got %d: %v", len(cfg.Skip), cfg.Skip)
	}
}

func TestMerge_CLISkipEmpty(t *testing.T) {
	cfg := &Config{Skip: []string{"clean"}}
	cfg.Merge("", nil)
	if len(cfg.Skip) != 1 {
		t.Errorf("expected 1 skip step, got %d", len(cfg.Skip))
	}
}

func TestHasSkip_True(t *testing.T) {
	cfg := &Config{Skip: []string{"test", "clean"}}
	if !cfg.HasSkip("test") {
		t.Error("expected HasSkip('test') to be true")
	}
}

func TestHasSkip_False(t *testing.T) {
	cfg := &Config{Skip: []string{"test", "clean"}}
	if cfg.HasSkip("build") {
		t.Error("expected HasSkip('build') to be false")
	}
}
