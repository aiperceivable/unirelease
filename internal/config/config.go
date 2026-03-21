package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the merged configuration.
type Config struct {
	Type      string         `toml:"type"`
	TagPrefix string         `toml:"tag_prefix"`
	Skip      []string       `toml:"skip"`
	Hooks     HooksConfig    `toml:"hooks"`
	Commands  CommandsConfig `toml:"commands"`
}

// HooksConfig holds pre/post hook commands.
type HooksConfig struct {
	PreBuild    string `toml:"pre_build"`
	PostBuild   string `toml:"post_build"`
	PrePublish  string `toml:"pre_publish"`
	PostPublish string `toml:"post_publish"`
}

// CommandsConfig holds command overrides.
type CommandsConfig struct {
	Build string `toml:"build"`
	Test  string `toml:"test"`
	Clean string `toml:"clean"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		TagPrefix: "v",
	}
}

// Load reads .unirelease.toml from the project directory.
// If the file does not exist, returns Default() with no error.
func Load(projectDir string) (*Config, error) {
	path := filepath.Join(projectDir, ".unirelease.toml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse .unirelease.toml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf(".unirelease.toml: %w", err)
	}

	return cfg, nil
}

// Merge applies CLI flag overrides to the config.
func (c *Config) Merge(cliType string) {
	if cliType != "" {
		c.Type = cliType
	}
}

// HasSkip checks if a step name is in the skip list.
func (c *Config) HasSkip(stepName string) bool {
	for _, s := range c.Skip {
		if s == stepName {
			return true
		}
	}
	return false
}

func (c *Config) validate() error {
	if c.Type != "" {
		valid := map[string]bool{"rust": true, "node": true, "bun": true, "python": true, "go": true}
		if !valid[c.Type] {
			return fmt.Errorf("invalid type %q; must be one of: rust, node, bun, python, go", c.Type)
		}
	}

	validSteps := map[string]bool{
		"detect": true, "read_version": true, "verify_env": true,
		"check_git_status": true, "clean": true, "build": true,
		"test": true, "verify": true, "git_tag": true, "github_release": true, "publish": true,
	}
	for _, step := range c.Skip {
		if !validSteps[step] {
			return fmt.Errorf("invalid skip step %q", step)
		}
	}

	return nil
}
