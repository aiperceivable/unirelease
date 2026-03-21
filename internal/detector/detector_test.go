package detector

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

func TestDetect_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeRust {
		t.Errorf("expected TypeRust, got %s", result.Type)
	}
	if result.Confidence != 100 {
		t.Errorf("expected confidence 100, got %d", result.Confidence)
	}
}

func TestDetect_Python(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]\nname = "test"\nversion = "0.1.0"`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypePython {
		t.Errorf("expected TypePython, got %s", result.Type)
	}
	if result.Confidence != 90 {
		t.Errorf("expected confidence 90, got %d", result.Confidence)
	}
}

func TestDetect_Node(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.0.0", "scripts": {"build": "tsc"}}`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeNode {
		t.Errorf("expected TypeNode, got %s", result.Type)
	}
	if result.Confidence != 50 {
		t.Errorf("expected confidence 50, got %d", result.Confidence)
	}
}

func TestDetect_BunBinary(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.0.0", "scripts": {"build": "bun build --compile src/index.ts --outfile dist/app"}}`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeBun {
		t.Errorf("expected TypeBun, got %s", result.Type)
	}
	if result.Confidence != 80 {
		t.Errorf("expected confidence 80, got %d", result.Confidence)
	}
}

func TestDetect_PriorityCargoOverNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"`)
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.0.0"}`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeRust {
		t.Errorf("expected TypeRust (higher priority), got %s", result.Type)
	}
}

func TestDetect_PriorityPythonOverNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", `[project]\nname = "test"\nversion = "0.1.0"`)
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.0.0"}`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypePython {
		t.Errorf("expected TypePython (higher priority), got %s", result.Type)
	}
}

func TestDetect_TypeOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"`)

	result, err := Detect(dir, "python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypePython {
		t.Errorf("expected TypePython (override), got %s", result.Type)
	}
	if result.Confidence != 1000 {
		t.Errorf("expected confidence 1000 for override, got %d", result.Confidence)
	}
}

func TestDetect_NoManifest(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect(dir, "")
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if err != ErrNoProject {
		t.Errorf("expected ErrNoProject, got %v", err)
	}
}

func TestDetect_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeGo {
		t.Errorf("expected TypeGo, got %s", result.Type)
	}
	if result.Confidence != 95 {
		t.Errorf("expected confidence 95, got %d", result.Confidence)
	}
}

func TestDetect_PriorityRustOverGo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"`)
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeRust {
		t.Errorf("expected TypeRust (higher priority), got %s", result.Type)
	}
}

func TestDetect_PriorityGoOverPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")
	writeFile(t, dir, "pyproject.toml", `[project]\nname = "test"\nversion = "0.1.0"`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeGo {
		t.Errorf("expected TypeGo (higher priority), got %s", result.Type)
	}
}

func TestDetect_PriorityGoOverNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module github.com/test/app\n\ngo 1.22\n")
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.0.0"}`)

	result, err := Detect(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeGo {
		t.Errorf("expected TypeGo (higher priority), got %s", result.Type)
	}
}

func TestDetect_GoTypeOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", `[package]\nname = "test"\nversion = "0.1.0"`)

	result, err := Detect(dir, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Type != TypeGo {
		t.Errorf("expected TypeGo (override), got %s", result.Type)
	}
}

func TestDetect_InvalidTypeOverride(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect(dir, "java")
	if err == nil {
		t.Fatal("expected error for invalid type override")
	}
}
