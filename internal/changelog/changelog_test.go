package changelog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtract_BasicFormat(t *testing.T) {
	dir := t.TempDir()
	content := `# Changelog

## [1.2.0] - 2026-03-21

### Added
- New feature X
- New feature Y

### Fixed
- Bug Z

## [1.1.0] - 2026-03-01

### Added
- Initial release
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	notes := Extract(dir, "1.2.0")
	if notes == "" {
		t.Fatal("expected non-empty notes")
	}
	if !contains(notes, "New feature X") {
		t.Errorf("expected notes to contain 'New feature X', got: %s", notes)
	}
	if !contains(notes, "Bug Z") {
		t.Errorf("expected notes to contain 'Bug Z', got: %s", notes)
	}
	if contains(notes, "Initial release") {
		t.Error("notes should not contain content from 1.1.0 section")
	}
}

func TestExtract_WithVPrefix(t *testing.T) {
	dir := t.TempDir()
	content := `# Changelog

## v2.0.0

- Breaking changes

## v1.0.0

- First release
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	notes := Extract(dir, "2.0.0")
	if notes == "" {
		t.Fatal("expected non-empty notes for v-prefix format")
	}
	if !contains(notes, "Breaking changes") {
		t.Errorf("expected 'Breaking changes', got: %s", notes)
	}
}

func TestExtract_VersionNotFound(t *testing.T) {
	dir := t.TempDir()
	content := `# Changelog

## [1.0.0]

- Something
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	notes := Extract(dir, "9.9.9")
	if notes != "" {
		t.Errorf("expected empty notes for missing version, got: %s", notes)
	}
}

func TestExtract_NoChangelog(t *testing.T) {
	dir := t.TempDir()
	notes := Extract(dir, "1.0.0")
	if notes != "" {
		t.Errorf("expected empty notes when no CHANGELOG.md, got: %s", notes)
	}
}

func TestFormatReleaseBody_WithChangelog(t *testing.T) {
	dir := t.TempDir()
	content := `## [1.0.0]

- Feature A
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	body := FormatReleaseBody(dir, "1.0.0")
	if !contains(body, "Feature A") {
		t.Errorf("expected body to contain 'Feature A', got: %s", body)
	}
}

func TestFormatReleaseBody_Fallback(t *testing.T) {
	dir := t.TempDir()
	body := FormatReleaseBody(dir, "1.0.0")
	if body != "Release version 1.0.0" {
		t.Errorf("expected fallback message, got: %s", body)
	}
}

func TestExtract_BracketAndBareVersions(t *testing.T) {
	dir := t.TempDir()
	content := `## 3.0.0

- Bare heading style
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	notes := Extract(dir, "3.0.0")
	if !contains(notes, "Bare heading style") {
		t.Errorf("expected notes for bare heading, got: %s", notes)
	}
}

func TestExtract_LastSection(t *testing.T) {
	dir := t.TempDir()
	content := `# Changelog

## [1.0.0]

- Only version
`
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)

	notes := Extract(dir, "1.0.0")
	if !contains(notes, "Only version") {
		t.Errorf("expected notes for last section, got: %s", notes)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
