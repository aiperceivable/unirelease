package providers

import (
	"testing"
)

func TestRustProvider_Name(t *testing.T) {
	p := &RustProvider{}
	if p.Name() != "rust" {
		t.Errorf("expected 'rust', got %q", p.Name())
	}
}

func TestRustProvider_PublishTarget(t *testing.T) {
	p := &RustProvider{}
	if p.PublishTarget() != "crates.io" {
		t.Errorf("expected 'crates.io', got %q", p.PublishTarget())
	}
}

func TestRustProvider_BinaryAssets_ReturnsNil(t *testing.T) {
	p := &RustProvider{}
	assets, err := p.BinaryAssets(nil)
	if err != nil || assets != nil {
		t.Errorf("expected nil, nil; got %v, %v", assets, err)
	}
}

func TestForType_Rust(t *testing.T) {
	p, err := ForType("rust")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "rust" {
		t.Errorf("expected 'rust', got %q", p.Name())
	}
}

func TestForType_Node(t *testing.T) {
	p, err := ForType("node")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "node" {
		t.Errorf("expected 'node', got %q", p.Name())
	}
}

func TestForType_Bun(t *testing.T) {
	p, err := ForType("bun")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "bun" {
		t.Errorf("expected 'bun', got %q", p.Name())
	}
}

func TestForType_Python(t *testing.T) {
	p, err := ForType("python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "python" {
		t.Errorf("expected 'python', got %q", p.Name())
	}
}

func TestForType_Invalid(t *testing.T) {
	_, err := ForType("java")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}
