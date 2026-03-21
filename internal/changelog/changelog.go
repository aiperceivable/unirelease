package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Extract reads CHANGELOG.md and returns the release notes for the given version.
// It looks for a section starting with "## [VERSION]" or "## VERSION" and extracts
// all content until the next "## " heading.
// Returns empty string (no error) if CHANGELOG.md doesn't exist or version not found.
func Extract(projectDir string, version string) string {
	path := filepath.Join(projectDir, "CHANGELOG.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractFromContent(string(data), version)
}

func extractFromContent(content string, version string) string {
	lines := strings.Split(content, "\n")

	// Match patterns: "## [1.2.3]", "## 1.2.3", "## [v1.2.3]", "## v1.2.3"
	patterns := []string{
		fmt.Sprintf("## [%s]", version),
		fmt.Sprintf("## %s", version),
		fmt.Sprintf("## [v%s]", version),
		fmt.Sprintf("## v%s", version),
	}

	found := false
	var result []string

	for _, line := range lines {
		if found {
			trimmed := strings.TrimSpace(line)
			// Stop at next heading
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			result = append(result, line)
		} else {
			trimmed := strings.TrimSpace(line)
			for _, pat := range patterns {
				if strings.HasPrefix(trimmed, pat) {
					found = true
					break
				}
			}
		}
	}

	if !found {
		return ""
	}

	// Trim leading/trailing blank lines
	text := strings.Join(result, "\n")
	text = strings.TrimSpace(text)
	return text
}

// FormatReleaseBody returns release notes for a GitHub Release.
// Falls back to a generic message if no CHANGELOG entry found.
func FormatReleaseBody(projectDir string, version string) string {
	notes := Extract(projectDir, version)
	if notes != "" {
		return notes
	}
	return fmt.Sprintf("Release version %s", version)
}
