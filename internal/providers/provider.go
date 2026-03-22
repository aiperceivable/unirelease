package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

// ForType returns the provider for a given project type string.
func ForType(projectType string) (pipeline.Provider, error) {
	switch projectType {
	case "rust":
		return &RustProvider{}, nil
	case "node":
		return &NodeProvider{}, nil
	case "bun":
		return &BunProvider{}, nil
	case "python":
		return &PythonProvider{}, nil
	case "go":
		return &GoProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}
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
