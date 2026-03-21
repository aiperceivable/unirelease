package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

// --- Rust Provider Operation Tests ---

func TestRustClean_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &RustProvider{}
	// dry-run mode: should not error
	if err := p.Clean(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRustBuild_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &RustProvider{}
	if err := p.Build(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRustTest_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &RustProvider{}
	if err := p.Test(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRustPublish_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &RustProvider{}
	if err := p.Publish(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Node Provider Operation Tests ---

func TestNodeClean_RemovesDirs(t *testing.T) {
	dir := t.TempDir()
	// Create dirs that should be cleaned
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.MkdirAll(filepath.Join(dir, "node_modules", ".cache"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "index.js"), []byte("test"), 0644)

	ctx, _ := newMockContext(dir)
	p := &NodeProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Error("expected dist/ to be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", ".cache")); !os.IsNotExist(err) {
		t.Error("expected node_modules/.cache/ to be removed")
	}
}

func TestNodeBuild_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &NodeProvider{}
	if err := p.Build(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNodePublish_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &NodeProvider{}
	if err := p.Publish(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Bun Provider Operation Tests ---

func TestBunClean_RemovesDist(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "app"), []byte("binary"), 0755)

	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Error("expected dist/ to be removed")
	}
}

func TestBunBuild_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
	if err := p.Build(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBunTest_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
	if err := p.Test(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBunBinaryAssets_OutfileFlag(t *testing.T) {
	dir := t.TempDir()
	// Create package.json with --outfile
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"build": "bun build --compile src/index.ts --outfile dist/myapp"}
	}`), 0644)
	// Create the output file
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "myapp"), []byte("binary"), 0755)

	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
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

func TestBunBinaryAssets_ScanDist(t *testing.T) {
	dir := t.TempDir()
	// Create package.json without --outfile
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"build": "bun build --compile src/index.ts"}
	}`), 0644)
	// Create executable in dist/
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.WriteFile(filepath.Join(dir, "dist", "myapp"), []byte("binary"), 0755)

	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
	assets, err := p.BinaryAssets(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assets) == 0 {
		t.Fatal("expected at least 1 asset")
	}
}

func TestBunBinaryAssets_NoDist(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"build": "bun build --compile src/index.ts"}
	}`), 0644)

	ctx, _ := newMockContext(dir)
	p := &BunProvider{}
	_, err := p.BinaryAssets(ctx)
	if err == nil {
		t.Fatal("expected error when dist/ doesn't exist")
	}
}

// --- Python Provider Operation Tests ---

func TestPythonClean_RemovesDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755)
	os.MkdirAll(filepath.Join(dir, "build"), 0755)
	os.MkdirAll(filepath.Join(dir, "foo.egg-info"), 0755)

	ctx, _ := newMockContext(dir)
	p := &PythonProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, d := range []string{"dist", "build", "foo.egg-info"} {
		if _, err := os.Stat(filepath.Join(dir, d)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", d)
		}
	}
}

func TestPythonClean_SrcLayout(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src", "foo.egg-info"), 0755)

	ctx, _ := newMockContext(dir)
	p := &PythonProvider{}
	if err := p.Clean(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "src", "foo.egg-info")); !os.IsNotExist(err) {
		t.Error("expected src/foo.egg-info to be removed")
	}
}

func TestPythonBuild_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &PythonProvider{}
	if err := p.Build(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPythonPublish_NoDistFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dist"), 0755) // empty dist/

	ctx, _ := newMockContext(dir)
	ctx.DryRun = false // need real mode to test the glob
	p := &PythonProvider{}
	err := p.Publish(ctx)
	if err == nil {
		t.Fatal("expected error for empty dist/")
	}
}

func TestPythonBinaryAssets_ReturnsNil(t *testing.T) {
	p := &PythonProvider{}
	assets, err := p.BinaryAssets(nil)
	if err != nil || assets != nil {
		t.Errorf("expected nil, nil; got %v, %v", assets, err)
	}
}

// --- Bun Publish returns ErrNoPublish ---

func TestBunPublish_ReturnsErrNoPublish(t *testing.T) {
	p := &BunProvider{}
	err := p.Publish(nil)
	if err != pipeline.ErrNoPublish {
		t.Errorf("expected ErrNoPublish, got %v", err)
	}
}

// --- Verify step tests ---

func TestRustVerify_ReturnsErrNoPublish(t *testing.T) {
	p := &RustProvider{}
	err := p.Verify(nil)
	if err != pipeline.ErrNoPublish {
		t.Errorf("expected ErrNoPublish, got %v", err)
	}
}

func TestNodeVerify_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &NodeProvider{}
	if err := p.Verify(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBunVerify_ReturnsErrNoPublish(t *testing.T) {
	p := &BunProvider{}
	err := p.Verify(nil)
	if err != pipeline.ErrNoPublish {
		t.Errorf("expected ErrNoPublish, got %v", err)
	}
}

func TestPythonVerify_DryRun(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := newMockContext(dir)
	p := &PythonProvider{}
	if err := p.Verify(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- RegistryCheck tests ---

func TestBunRegistryCheck_ReturnsFalse(t *testing.T) {
	p := &BunProvider{}
	exists, err := p.RegistryCheck(nil)
	if err != nil || exists {
		t.Errorf("expected false, nil; got %v, %v", exists, err)
	}
}
