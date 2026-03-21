package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aipartnerup/unirelease/internal/pipeline"
)

// BunProvider implements Provider for Bun binary projects.
type BunProvider struct{}

func (p *BunProvider) Name() string { return "bun" }

func (p *BunProvider) VerifyEnv() ([]string, error) {
	if _, err := exec.LookPath("bun"); err != nil {
		return []string{"bun"}, fmt.Errorf("bun not found; install from https://bun.sh")
	}
	return nil, nil
}

func (p *BunProvider) Clean(ctx *pipeline.Context) error {
	dir := filepath.Join(ctx.ProjectDir, "dist")
	return os.RemoveAll(dir)
}

func (p *BunProvider) Build(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("bun", "run", "build")
	return err
}

func (p *BunProvider) Test(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("bun", "test")
	return err
}

func (p *BunProvider) Verify(ctx *pipeline.Context) error {
	return pipeline.ErrNoPublish // no package verification for binary projects
}

func (p *BunProvider) Publish(ctx *pipeline.Context) error {
	return pipeline.ErrNoPublish
}

func (p *BunProvider) RegistryCheck(ctx *pipeline.Context) (bool, error) {
	return false, nil // bun binaries are published via GitHub Release, no registry
}

func (p *BunProvider) PublishTarget() string { return "GitHub Release" }

func (p *BunProvider) BinaryAssets(ctx *pipeline.Context) ([]string, error) {
	// Strategy 1: Parse --outfile from build script
	pkgPath := filepath.Join(ctx.ProjectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("read package.json for binary assets: %w", err)
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json for binary assets: %w", err)
	}

	for _, script := range pkg.Scripts {
		if strings.Contains(script, "bun build --compile") {
			parts := strings.Fields(script)
			for i, part := range parts {
				if part == "--outfile" && i+1 < len(parts) {
					outfile := parts[i+1]
					absPath := filepath.Join(ctx.ProjectDir, outfile)
					if _, err := os.Stat(absPath); err == nil {
						return []string{absPath}, nil
					}
				}
			}
		}
	}

	// Strategy 2: Scan dist/ for executable files
	distDir := filepath.Join(ctx.ProjectDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return nil, fmt.Errorf("no binary found: dist/ not found and --outfile not parsed")
	}
	var assets []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue // skip entries we can't stat
		}
		if info.Mode()&0111 != 0 || filepath.Ext(entry.Name()) == "" {
			assets = append(assets, filepath.Join(distDir, entry.Name()))
		}
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("no binary assets found in dist/")
	}
	return assets, nil
}
