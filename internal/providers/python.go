package providers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aiperceivable/unirelease/internal/pipeline"
)

// PythonProvider implements Provider for Python projects.
type PythonProvider struct{}

func (p *PythonProvider) Name() string { return "python" }

// resolvePythonCommand returns "python3" or "python", whichever is available.
func resolvePythonCommand() string {
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	if _, err := exec.LookPath("python"); err == nil {
		return "python"
	}
	return ""
}

func (p *PythonProvider) VerifyEnv() ([]string, error) {
	var missing []string

	pythonCmd := resolvePythonCommand()
	if pythonCmd == "" {
		missing = append(missing, "python/python3")
	}

	if _, err := exec.LookPath("twine"); err != nil {
		missing = append(missing, "twine (pip install twine)")
	}

	if pythonCmd != "" {
		cmd := exec.Command(pythonCmd, "-m", "build", "--version")
		if err := cmd.Run(); err != nil {
			missing = append(missing, "build (pip install build)")
		}
	}

	if len(missing) > 0 {
		return missing, fmt.Errorf("missing tools: %v", missing)
	}
	return nil, nil
}

func (p *PythonProvider) Clean(ctx *pipeline.Context) error {
	for _, dir := range []string{"dist", "build", ".eggs"} {
		path := filepath.Join(ctx.ProjectDir, dir)
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("clean %s: %w", dir, err)
		}
	}

	matches, err := filepath.Glob(filepath.Join(ctx.ProjectDir, "*.egg-info"))
	if err != nil {
		return fmt.Errorf("glob egg-info: %w", err)
	}
	for _, match := range matches {
		if err := os.RemoveAll(match); err != nil {
			return fmt.Errorf("clean %s: %w", match, err)
		}
	}

	srcMatches, err := filepath.Glob(filepath.Join(ctx.ProjectDir, "src", "*.egg-info"))
	if err != nil {
		return fmt.Errorf("glob src egg-info: %w", err)
	}
	for _, match := range srcMatches {
		if err := os.RemoveAll(match); err != nil {
			return fmt.Errorf("clean %s: %w", match, err)
		}
	}

	ctx.UI.Info("Cleaned dist/, build/, *.egg-info/")
	return nil
}

func (p *PythonProvider) Build(ctx *pipeline.Context) error {
	pythonCmd := resolvePythonCommand()
	if pythonCmd == "" {
		return fmt.Errorf("python not found")
	}
	_, err := ctx.Runner.Run(pythonCmd, "-m", "build")
	return err
}

func (p *PythonProvider) Test(ctx *pipeline.Context) error {
	if _, err := exec.LookPath("pytest"); err == nil {
		_, err := ctx.Runner.Run("pytest")
		return err
	}
	pythonCmd := resolvePythonCommand()
	if pythonCmd == "" {
		return fmt.Errorf("python not found")
	}
	_, err := ctx.Runner.Run(pythonCmd, "-m", "pytest")
	return err
}

func (p *PythonProvider) Publish(ctx *pipeline.Context) error {
	distDir := filepath.Join(ctx.ProjectDir, "dist")
	patterns := []string{
		filepath.Join(distDir, "*.whl"),
		filepath.Join(distDir, "*.tar.gz"),
	}
	var files []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	if len(files) == 0 {
		return fmt.Errorf("no distribution files in dist/; run build first")
	}

	args := append([]string{"upload"}, files...)
	_, err := ctx.Runner.Run("twine", args...)
	return err
}

func (p *PythonProvider) Verify(ctx *pipeline.Context) error {
	distDir := filepath.Join(ctx.ProjectDir, "dist")
	patterns := []string{
		filepath.Join(distDir, "*.whl"),
		filepath.Join(distDir, "*.tar.gz"),
	}
	var files []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	if len(files) == 0 {
		return fmt.Errorf("no distribution files in dist/; run build first")
	}
	args := append([]string{"check"}, files...)
	_, err := ctx.Runner.Run("twine", args...)
	return err
}

func (p *PythonProvider) PublishTarget() string { return "PyPI" }

func (p *PythonProvider) BinaryAssets(ctx *pipeline.Context) ([]string, error) {
	return nil, nil
}

// RegistryCheck checks if the version exists on PyPI.
func (p *PythonProvider) RegistryCheck(ctx *pipeline.Context) (bool, error) {
	name := readPyprojectName(ctx.ProjectDir)
	if name == "" {
		return false, nil
	}
	cmd := exec.Command("pip", "index", "versions", name)
	out, err := cmd.Output()
	if err != nil {
		return false, nil // pip may not support index, or package doesn't exist
	}
	// Output: "foo (1.0.0)\nAvailable versions: 1.0.0, 0.9.0"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Available versions:") {
			versions := strings.TrimPrefix(line, "Available versions:")
			for _, v := range strings.Split(versions, ",") {
				if strings.TrimSpace(v) == ctx.Version {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func readPyprojectName(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, "pyproject.toml"))
	if err != nil {
		return ""
	}
	var pyproject struct {
		Project struct {
			Name string `toml:"name"`
		} `toml:"project"`
	}
	if toml.Unmarshal(data, &pyproject) != nil {
		return ""
	}
	return pyproject.Project.Name
}
