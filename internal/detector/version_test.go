package detector

import (
	"testing"
)

func TestReadVersion_Cargo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.3.0\"\n")

	ver, err := ReadVersion(dir, TypeRust, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "0.3.0" {
		t.Errorf("expected 0.3.0, got %s", ver)
	}
}

func TestReadVersion_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "version": "1.2.0"}`)

	ver, err := ReadVersion(dir, TypeNode, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.0" {
		t.Errorf("expected 1.2.0, got %s", ver)
	}
}

func TestReadVersion_Pyproject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"test\"\nversion = \"2.0.0\"\n")

	ver, err := ReadVersion(dir, TypePython, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", ver)
	}
}

func TestReadVersion_Override(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\nversion = \"0.1.0\"\n")

	ver, err := ReadVersion(dir, TypeRust, "9.9.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "9.9.9" {
		t.Errorf("expected 9.9.9, got %s", ver)
	}
}

func TestReadVersion_BunUsesPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "version": "3.0.0"}`)

	ver, err := ReadVersion(dir, TypeBun, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "3.0.0" {
		t.Errorf("expected 3.0.0, got %s", ver)
	}
}

func TestReadVersion_MissingVersion_Cargo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\n")

	_, err := ReadVersion(dir, TypeRust, "")
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestReadVersion_MissingVersion_JSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)

	_, err := ReadVersion(dir, TypeNode, "")
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestReadVersion_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "1.5.0")

	ver, err := ReadVersion(dir, TypeGo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.5.0" {
		t.Errorf("expected 1.5.0, got %s", ver)
	}
}

func TestReadVersion_Go_VPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "v2.0.0")

	ver, err := ReadVersion(dir, TypeGo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "2.0.0" {
		t.Errorf("expected 2.0.0 (v stripped), got %s", ver)
	}
}

func TestReadVersion_Go_Whitespace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "  1.0.0\n")

	ver, err := ReadVersion(dir, TypeGo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", ver)
	}
}

func TestReadVersion_Go_NoVersionFile(t *testing.T) {
	dir := t.TempDir()

	_, err := ReadVersion(dir, TypeGo, "")
	if err == nil {
		t.Fatal("expected error for missing VERSION file")
	}
}

func TestReadVersion_Go_EmptyVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "VERSION", "")

	_, err := ReadVersion(dir, TypeGo, "")
	if err == nil {
		t.Fatal("expected error for empty VERSION file")
	}
}

func TestReadVersion_MalformedTOML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "this is not valid toml {{{{")

	_, err := ReadVersion(dir, TypeRust, "")
	if err == nil {
		t.Fatal("expected error for malformed TOML")
	}
}

func TestReadVersion_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "not json at all")

	_, err := ReadVersion(dir, TypeNode, "")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
