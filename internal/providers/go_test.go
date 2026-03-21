package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

func TestGoProvider_Name(t *testing.T) {
	p := &GoProvider{}
	if p.Name() != "go" {
		t.Errorf("expected 'go', got %q", p.Name())
	}
}

func TestGoProvider_PublishTarget(t *testing.T) {
	p := &GoProvider{}
	if p.PublishTarget() != "GitHub Release" {
		t.Errorf("expected 'GitHub Release', got %q", p.PublishTarget())
	}
}

func TestGoProvider_Publish_ReturnsErrNoPublish(t *testing.T) {
	p := &GoProvider{}
	err := p.Publish(nil)
	if err != pipeline.ErrNoPublish {
		t.Errorf("expected ErrNoPublish, got %v", err)
	}
}

func TestForType_Go(t *testing.T) {
	p, err := ForType("go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "go" {
		t.Errorf("expected 'go', got %q", p.Name())
	}
}

func TestGoClean_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGoClean_RemovesDist(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "myapp"), []byte("binary"), 0755)

	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Error("expected dist/ to be removed")
	}
}

func TestGoBuild_DryRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/myapp\n\ngo 1.22\n"), 0644)
	ctx, _ := newMockContext(dir)
	ctx.Version = "1.0.0"
	p := &GoProvider{}
	if err := p.Build(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGoTest_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	if err := p.Test(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGoBinaryAssets_WithFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "myapp"), []byte("binary"), 0755)

	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	assets, err := p.BinaryAssets(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if filepath.Base(assets[0]) != "myapp" {
		t.Errorf("expected 'myapp', got %q", filepath.Base(assets[0]))
	}
}

func TestGoBinaryAssets_NoDist(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	_, err := p.BinaryAssets(ctx)
	if err == nil {
		t.Fatal("expected error when dist/ doesn't exist")
	}
}

func TestGoBinaryName_FromGoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/user/myapp\n\ngo 1.22\n"), 0644)
	name := goBinaryName(dir)
	if name != "myapp" {
		t.Errorf("expected 'myapp', got %q", name)
	}
}

func TestGoBinaryName_MajorVersionSuffix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/user/mylib/v2\n\ngo 1.22\n"), 0644)
	name := goBinaryName(dir)
	if name != "mylib" {
		t.Errorf("expected 'mylib', got %q", name)
	}
}

func TestGoBinaryName_V3Suffix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/user/mylib/v3\n\ngo 1.22\n"), 0644)
	name := goBinaryName(dir)
	if name != "mylib" {
		t.Errorf("expected 'mylib', got %q", name)
	}
}

func TestGoBinaryAssets_SkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "myapp"), []byte("binary"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", ".DS_Store"), []byte("junk"), 0644)

	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	assets, err := p.BinaryAssets(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset (hidden files filtered), got %d", len(assets))
	}
	if filepath.Base(assets[0]) != "myapp" {
		t.Errorf("expected 'myapp', got %q", filepath.Base(assets[0]))
	}
}

func TestGoVerify_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &GoProvider{}
	if err := p.Verify(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGoRegistryCheck_ReturnsFalse(t *testing.T) {
	p := &GoProvider{}
	exists, err := p.RegistryCheck(nil)
	if err != nil || exists {
		t.Errorf("expected false, nil; got %v, %v", exists, err)
	}
}

func TestGoBinaryName_FallbackToDirName(t *testing.T) {
	dir := t.TempDir()
	name := goBinaryName(dir)
	if name != filepath.Base(dir) {
		t.Errorf("expected %q, got %q", filepath.Base(dir), name)
	}
}
