package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aiperceivable/unirelease/internal/pipeline"
)

// NodeProvider implements Provider for Node.js projects.
type NodeProvider struct {
	packageManager string
}

func (p *NodeProvider) Name() string { return "node" }

// detectPackageManager determines the package manager from lockfile presence.
func detectPackageManager(projectDir string) string {
	lockfiles := []struct {
		file    string
		manager string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"bun.lockb", "bun"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, lf := range lockfiles {
		if _, err := os.Stat(filepath.Join(projectDir, lf.file)); err == nil {
			return lf.manager
		}
	}
	return "npm"
}

func (p *NodeProvider) VerifyEnv() ([]string, error) {
	var missing []string
	// Check that npm is available (needed for publish)
	if _, err := exec.LookPath("npm"); err != nil {
		missing = append(missing, "npm")
	}
	if len(missing) > 0 {
		return missing, fmt.Errorf("missing tools: %v", missing)
	}
	return nil, nil
}

func (p *NodeProvider) Clean(ctx *pipeline.Context) error {
	dirs := []string{
		filepath.Join(ctx.ProjectDir, "dist"),
		filepath.Join(ctx.ProjectDir, "node_modules", ".cache"),
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("clean %s: %w", dir, err)
		}
	}
	ctx.UI.Info("Cleaned dist/ and node_modules/.cache/")
	return nil
}

func (p *NodeProvider) Build(ctx *pipeline.Context) error {
	pm := p.resolvePackageManager(ctx.ProjectDir)
	args := []string{"build"}
	if pm == "npm" {
		args = []string{"run", "build"}
	}
	_, err := ctx.Runner.Run(pm, args...)
	return err
}

func (p *NodeProvider) Test(ctx *pipeline.Context) error {
	pm := p.resolvePackageManager(ctx.ProjectDir)
	args := []string{"test"}
	if pm == "npm" {
		args = []string{"run", "test"}
	}
	_, err := ctx.Runner.Run(pm, args...)
	return err
}

func (p *NodeProvider) Publish(ctx *pipeline.Context) error {
	args := []string{"publish", "--access", "public"}
	var maskIndices []int
	if ctx.OTP != "" {
		args = append(args, "--otp", ctx.OTP)
		maskIndices = append(maskIndices, len(args)-1)
	}
	args = append(args, ctx.PublishArgs...)
	_, err := ctx.Runner.RunSensitive(maskIndices, "npm", args...)
	return err
}

func (p *NodeProvider) Verify(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("npm", "pack", "--dry-run")
	return err
}

func (p *NodeProvider) PublishTarget() string { return "npm" }

func (p *NodeProvider) BinaryAssets(ctx *pipeline.Context) ([]string, error) {
	return nil, nil
}

// RegistryCheck checks if the version exists on npm registry.
func (p *NodeProvider) RegistryCheck(ctx *pipeline.Context) (bool, error) {
	name := readPackageName(ctx.ProjectDir)
	if name == "" {
		return false, nil
	}
	cmd := exec.Command("npm", "view", name, "versions", "--json")
	out, err := cmd.Output()
	if err != nil {
		return false, nil // package may not exist yet
	}
	var versions []string
	if json.Unmarshal(out, &versions) != nil {
		// npm returns a bare string for single-version packages
		var single string
		if json.Unmarshal(out, &single) == nil {
			versions = []string{single}
		}
	}
	for _, v := range versions {
		if strings.TrimSpace(v) == ctx.Version {
			return true, nil
		}
	}
	return false, nil
}

func readPackageName(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return pkg.Name
}

func (p *NodeProvider) resolvePackageManager(projectDir string) string {
	if p.packageManager == "" {
		p.packageManager = detectPackageManager(projectDir)
	}
	return p.packageManager
}
