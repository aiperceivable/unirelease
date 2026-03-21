package detector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectType represents a detected project type.
type ProjectType string

const (
	TypeRust   ProjectType = "rust"
	TypeNode   ProjectType = "node"
	TypeBun    ProjectType = "bun"
	TypePython ProjectType = "python"
	TypeGo     ProjectType = "go"
)

// ValidTypes returns all valid project types for flag validation.
func ValidTypes() []ProjectType {
	return []ProjectType{TypeRust, TypeNode, TypeBun, TypePython, TypeGo}
}

// DetectionResult holds the result of project type detection.
type DetectionResult struct {
	Type       ProjectType
	Confidence int
	Manifest   string
}

// ErrNoProject is returned when no supported manifest file is found.
var ErrNoProject = errors.New("no supported project manifest found")

// Detect scans the project directory and returns the detected project type.
// If typeOverride is non-empty, it is used directly (from --type flag).
func Detect(projectDir string, typeOverride string) (*DetectionResult, error) {
	if typeOverride != "" {
		pt := ProjectType(typeOverride)
		if !isValidType(pt) {
			return nil, fmt.Errorf("unsupported project type: %q (valid: %s)", typeOverride, validTypesList())
		}
		return &DetectionResult{Type: pt, Confidence: 1000, Manifest: ""}, nil
	}

	var candidates []DetectionResult

	// Check Cargo.toml → Rust
	if fileExists(filepath.Join(projectDir, "Cargo.toml")) {
		candidates = append(candidates, DetectionResult{
			Type:       TypeRust,
			Confidence: 100,
			Manifest:   "Cargo.toml",
		})
	}

	// Check pyproject.toml → Python
	if fileExists(filepath.Join(projectDir, "pyproject.toml")) {
		candidates = append(candidates, DetectionResult{
			Type:       TypePython,
			Confidence: 90,
			Manifest:   "pyproject.toml",
		})
	}

	// Check go.mod → Go
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		candidates = append(candidates, DetectionResult{
			Type:       TypeGo,
			Confidence: 95,
			Manifest:   "go.mod",
		})
	}

	// Check package.json → Node or Bun
	pkgPath := filepath.Join(projectDir, "package.json")
	if fileExists(pkgPath) {
		isBun, err := isBunBinary(pkgPath)
		if err != nil {
			// If we can't parse package.json, treat as Node with lower confidence
			candidates = append(candidates, DetectionResult{
				Type:       TypeNode,
				Confidence: 50,
				Manifest:   "package.json",
			})
		} else if isBun {
			candidates = append(candidates, DetectionResult{
				Type:       TypeBun,
				Confidence: 80,
				Manifest:   "package.json",
			})
		} else {
			candidates = append(candidates, DetectionResult{
				Type:       TypeNode,
				Confidence: 50,
				Manifest:   "package.json",
			})
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoProject
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	return &candidates[0], nil
}

// isBunBinary checks if any script in package.json contains "bun build --compile".
func isBunBinary(packageJSONPath string) (bool, error) {
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false, err
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, err
	}
	for _, script := range pkg.Scripts {
		if strings.Contains(script, "bun build --compile") {
			return true, nil
		}
	}
	return false, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isValidType(pt ProjectType) bool {
	for _, valid := range ValidTypes() {
		if pt == valid {
			return true
		}
	}
	return false
}

func validTypesList() string {
	types := ValidTypes()
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strings.Join(strs, ", ")
}
