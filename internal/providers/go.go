package providers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

// GoProvider implements Provider for Go projects.
type GoProvider struct{}

func (p *GoProvider) Name() string { return "go" }

func (p *GoProvider) VerifyEnv() ([]string, error) {
	var missing []string
	if _, err := exec.LookPath("go"); err != nil {
		missing = append(missing, "go")
	}
	if len(missing) > 0 {
		return missing, fmt.Errorf("missing tools: %v; install Go from https://go.dev/dl", missing)
	}
	return nil, nil
}

func (p *GoProvider) Clean(ctx *pipeline.Context) error {
	if _, err := ctx.Runner.Run("go", "clean"); err != nil {
		return err
	}
	distDir := filepath.Join(ctx.ProjectDir, "dist")
	if err := os.RemoveAll(distDir); err != nil {
		return fmt.Errorf("clean dist/: %w", err)
	}
	ctx.UI.Info("Cleaned go cache and dist/")
	return nil
}

func (p *GoProvider) Build(ctx *pipeline.Context) error {
	// Build to dist/ directory. Use -trimpath for reproducible builds.
	outDir := filepath.Join(ctx.ProjectDir, "dist")
	if !ctx.DryRun {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("create dist/: %w", err)
		}
	}

	// Determine the binary name from the module path.
	binName := goBinaryName(ctx.ProjectDir)
	outPath := filepath.Join(outDir, binName)

	ldflags := fmt.Sprintf("-s -w -X main.version=%s", ctx.Version)
	_, err := ctx.Runner.Run("go", "build", "-trimpath", "-ldflags", ldflags, "-o", outPath, ".")
	return err
}

func (p *GoProvider) Test(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("go", "test", "./...")
	return err
}

func (p *GoProvider) Verify(ctx *pipeline.Context) error {
	// Go has go vet as a verification step
	_, err := ctx.Runner.Run("go", "vet", "./...")
	return err
}

func (p *GoProvider) Publish(ctx *pipeline.Context) error {
	// Go modules are published via git tags + proxy.golang.org.
	// The git_tag and github_release steps handle this.
	return pipeline.ErrNoPublish
}

func (p *GoProvider) PublishTarget() string { return "GitHub Release" }

func (p *GoProvider) BinaryAssets(ctx *pipeline.Context) ([]string, error) {
	distDir := filepath.Join(ctx.ProjectDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return nil, fmt.Errorf("read dist/: %w", err)
	}
	var assets []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip hidden files (e.g., .DS_Store)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		assets = append(assets, filepath.Join(distDir, entry.Name()))
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("no binary files found in dist/; run build first")
	}
	return assets, nil
}

func (p *GoProvider) RegistryCheck(ctx *pipeline.Context) (bool, error) {
	return false, nil // Go modules publish via git tags, no separate registry check
}

// goBinaryName extracts the binary name from go.mod module path.
// Falls back to the directory name if go.mod can't be parsed.
// Handles major version suffixes (e.g., github.com/foo/bar/v2 → bar).
func goBinaryName(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				mod := strings.TrimPrefix(line, "module ")
				mod = strings.TrimSpace(mod)
				parts := strings.Split(mod, "/")
				name := parts[len(parts)-1]
				// Skip major version suffixes like v2, v3, etc.
				if isMajorVersionSuffix(name) && len(parts) >= 2 {
					name = parts[len(parts)-2]
				}
				return name
			}
		}
	}
	return filepath.Base(projectDir)
}

// isMajorVersionSuffix returns true for "v2", "v3", etc.
func isMajorVersionSuffix(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
