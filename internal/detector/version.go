package detector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ReadVersion reads the version string from the project manifest.
// If versionOverride is non-empty, returns it directly (from --version flag).
func ReadVersion(projectDir string, projectType ProjectType, versionOverride string) (string, error) {
	if versionOverride != "" {
		return versionOverride, nil
	}

	switch projectType {
	case TypeRust:
		return readCargoVersion(projectDir)
	case TypeNode, TypeBun:
		return readPackageJSONVersion(projectDir)
	case TypePython:
		return readPyprojectVersion(projectDir)
	case TypeGo:
		return readGoVersion(projectDir)
	default:
		return "", fmt.Errorf("unsupported project type for version reading: %s", projectType)
	}
}

func readCargoVersion(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "Cargo.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read Cargo.toml: %w", err)
	}
	var cargo struct {
		Package struct {
			Version string `toml:"version"`
		} `toml:"package"`
	}
	if err := toml.Unmarshal(data, &cargo); err != nil {
		return "", fmt.Errorf("parse Cargo.toml: %w", err)
	}
	if cargo.Package.Version == "" {
		return "", fmt.Errorf("no version found in Cargo.toml [package] section")
	}
	return cargo.Package.Version, nil
}

func readPackageJSONVersion(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read package.json: %w", err)
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", fmt.Errorf("parse package.json: %w", err)
	}
	if pkg.Version == "" {
		return "", fmt.Errorf("no 'version' field in package.json")
	}
	return pkg.Version, nil
}

func readGoVersion(projectDir string) (string, error) {
	// Go projects don't embed version in go.mod. Check for a VERSION file.
	path := filepath.Join(projectDir, "VERSION")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no VERSION file found; Go projects require a VERSION file or --version flag")
	}
	ver := strings.TrimSpace(string(data))
	if ver == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}
	// Strip leading "v" prefix to avoid producing tags like "vv1.2.3"
	ver = strings.TrimPrefix(ver, "v")
	return ver, nil
}

func readPyprojectVersion(projectDir string) (string, error) {
	path := filepath.Join(projectDir, "pyproject.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read pyproject.toml: %w", err)
	}
	var pyproject struct {
		Project struct {
			Version string `toml:"version"`
		} `toml:"project"`
	}
	if err := toml.Unmarshal(data, &pyproject); err != nil {
		return "", fmt.Errorf("parse pyproject.toml: %w", err)
	}
	if pyproject.Project.Version == "" {
		return "", fmt.Errorf("no version found in pyproject.toml [project] section")
	}
	return pyproject.Project.Version, nil
}
