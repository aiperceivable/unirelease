package providers

import (
	"fmt"

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
