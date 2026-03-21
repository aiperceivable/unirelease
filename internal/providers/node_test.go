package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

func TestDetectPackageManager_Pnpm(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644)
	if pm := detectPackageManager(dir); pm != "pnpm" {
		t.Errorf("expected 'pnpm', got %q", pm)
	}
}

func TestDetectPackageManager_Bun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bun.lockb"), []byte(""), 0644)
	if pm := detectPackageManager(dir); pm != "bun" {
		t.Errorf("expected 'bun', got %q", pm)
	}
}

func TestDetectPackageManager_Yarn(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644)
	if pm := detectPackageManager(dir); pm != "yarn" {
		t.Errorf("expected 'yarn', got %q", pm)
	}
}

func TestDetectPackageManager_Npm(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(""), 0644)
	if pm := detectPackageManager(dir); pm != "npm" {
		t.Errorf("expected 'npm', got %q", pm)
	}
}

func TestDetectPackageManager_Default(t *testing.T) {
	dir := t.TempDir()
	if pm := detectPackageManager(dir); pm != "npm" {
		t.Errorf("expected 'npm' as default, got %q", pm)
	}
}

func TestDetectPackageManager_Priority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(""), 0644)
	if pm := detectPackageManager(dir); pm != "pnpm" {
		t.Errorf("expected 'pnpm' (higher priority), got %q", pm)
	}
}

func TestNodeProvider_PublishTarget(t *testing.T) {
	p := &NodeProvider{}
	if p.PublishTarget() != "npm" {
		t.Errorf("expected 'npm', got %q", p.PublishTarget())
	}
}

func TestNodeProvider_BinaryAssets_ReturnsNil(t *testing.T) {
	p := &NodeProvider{}
	assets, err := p.BinaryAssets(nil)
	if err != nil || assets != nil {
		t.Errorf("expected nil, nil; got %v, %v", assets, err)
	}
}

func TestBunProvider_Publish_ReturnsErrNoPublish(t *testing.T) {
	p := &BunProvider{}
	err := p.Publish(nil)
	if err != pipeline.ErrNoPublish {
		t.Errorf("expected ErrNoPublish, got %v", err)
	}
}

func TestBunProvider_PublishTarget(t *testing.T) {
	p := &BunProvider{}
	if p.PublishTarget() != "GitHub Release" {
		t.Errorf("expected 'GitHub Release', got %q", p.PublishTarget())
	}
}
