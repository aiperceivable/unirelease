# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.1.0] - 2026-03-21

### Added

- **CLI shell** with Cobra: `unirelease [path]` with `--dry-run`, `--yes`, `--step`, `--type`, `--set-version`, `--version` flags
- **Auto-detection** of project type from manifest files (Cargo.toml, go.mod, package.json, pyproject.toml)
- **Detection priority**: Rust (100) > Go (95) > Python (90) > Bun (80) > Node (50), overridable with `--type`
- **11-step pipeline**: detect → read_version → verify_env → check_git_status → clean → build → test → verify → git_tag → github_release → publish
- **Rust provider**: cargo build/test/clean/publish to crates.io
- **Go provider**: go build with -trimpath/-ldflags, go test, go vet, VERSION file support, binary upload to GitHub Release
- **Node.js provider**: auto-detect package manager from lockfiles (pnpm/bun/yarn/npm), npm publish
- **Bun provider**: bun build --compile, binary asset detection and upload to GitHub Release
- **Python provider**: python -m build, twine check/upload to PyPI, egg-info cleanup
- **Git operations**: clean working tree check, annotated tag creation, push to remote
- **GitHub Release**: create release via go-github SDK, CHANGELOG.md parsing for release notes, binary asset upload
- **GitHub auth**: 4-source token resolution (GITHUB_TOKEN → GH_TOKEN → gh CLI → git config)
- **Registry pre-check**: check crates.io/npm/PyPI for existing version before publish, prompt to confirm
- **Package verification**: npm pack --dry-run (Node), twine check (Python), go vet (Go)
- **Configuration**: `.unirelease.toml` with type override, tag_prefix, skip steps, custom commands, pre/post hooks
- **Interactive prompts**: confirmation before destructive steps (git_tag, github_release, publish), `--yes` to skip
- **Dry-run mode**: preview all steps without executing (`--dry-run`)
- **Single-step mode**: run only a specific step (`--step build`)
- **Summary report**: step results + remote status checks (git tag, GitHub Release, registry) after pipeline
- **Colored output**: step progress, success/warning/error messages, boxed summary
- **Cross-platform**: Windows (cmd /c) and Unix (sh -c) command execution
- **183 tests** across 10 packages: unit tests, git integration tests (temp bare repos), GitHub API tests (httptest mock), pipeline step tests, E2E dry-run tests for all 5 project types
