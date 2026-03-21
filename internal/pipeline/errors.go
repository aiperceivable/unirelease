package pipeline

import "errors"

// ErrNoPublish indicates the provider does not publish to a registry.
// The pipeline engine treats this as a skip, not an error.
var ErrNoPublish = errors.New("provider does not publish to a registry")

// ErrMissingTool indicates required tools are not installed.
var ErrMissingTool = errors.New("missing required tools")
