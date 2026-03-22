package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/aiperceivable/unirelease/internal/pipeline"
)

// RustProvider implements Provider for Rust/Cargo projects.
type RustProvider struct{}

func (p *RustProvider) Name() string { return "rust" }

func (p *RustProvider) VerifyEnv() ([]string, error) {
	var missing []string
	for _, tool := range []string{"cargo", "rustc"} {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return missing, fmt.Errorf("missing tools: %v; install Rust from https://rustup.rs", missing)
	}
	return nil, nil
}

func (p *RustProvider) Clean(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("cargo", "clean")
	return err
}

func (p *RustProvider) Build(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("cargo", "build", "--release")
	return err
}

func (p *RustProvider) Test(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("cargo", "test")
	return err
}

func (p *RustProvider) Publish(ctx *pipeline.Context) error {
	_, err := ctx.Runner.Run("cargo", "publish")
	return err
}

func (p *RustProvider) Verify(ctx *pipeline.Context) error {
	// Rust has no separate verify step; cargo build --release in build step is sufficient.
	return pipeline.ErrNoPublish
}

func (p *RustProvider) PublishTarget() string { return "crates.io" }

func (p *RustProvider) BinaryAssets(ctx *pipeline.Context) ([]string, error) {
	return nil, nil
}

// RegistryCheck checks if the version exists on crates.io.
func (p *RustProvider) RegistryCheck(ctx *pipeline.Context) (bool, error) {
	name := readCargoName(ctx.ProjectDir)
	if name == "" {
		return false, nil
	}
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s/%s", name, ctx.Version)
	resp, err := http.Get(url)
	if err != nil {
		return false, nil // network error, don't block
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var result struct {
			Version struct {
				Num string `json:"num"`
			} `json:"version"`
		}
		if json.NewDecoder(resp.Body).Decode(&result) == nil && result.Version.Num == ctx.Version {
			return true, nil
		}
	}
	return false, nil
}

func readCargoName(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "Cargo.toml"))
	if err != nil {
		return ""
	}
	var cargo struct {
		Package struct {
			Name string `toml:"name"`
		} `toml:"package"`
	}
	if toml.Unmarshal(data, &cargo) != nil {
		return ""
	}
	return cargo.Package.Name
}
