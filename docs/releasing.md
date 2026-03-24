# Releasing & Distribution

## TL;DR

```bash
# Release unirelease (one command)
./scripts/release.sh 0.2.0

# Or without GoReleaser (builds + uploads locally via gh CLI)
./scripts/release.sh 0.2.0 --local
```

That's it. The script handles tests, cross-compilation, VERSION file, git tag, push, and GitHub Release.

---

## Scripts

All release automation lives in `scripts/`:

| Script | Purpose | Usage |
|--------|---------|-------|
| `scripts/build.sh` | Build binaries | `./scripts/build.sh [--all] [version]` |
| `scripts/install.sh` | One-line installer for users | `curl -sSL .../install.sh \| bash` |
| `scripts/release.sh` | Full release pipeline | `./scripts/release.sh <version> [--local]` |

### scripts/build.sh

Build unirelease binaries:

```bash
./scripts/build.sh              # current platform only
./scripts/build.sh --all        # cross-compile all 5 platforms
./scripts/build.sh 1.0.0        # current platform with version
./scripts/build.sh --all 1.0.0  # all platforms with version

# Output:
# dist/unirelease-linux-amd64
# dist/unirelease-linux-arm64
# dist/unirelease-darwin-amd64
# dist/unirelease-darwin-arm64
# dist/unirelease-windows-amd64.exe
```

### scripts/install.sh

One-line installer for end users. Auto-detects OS and architecture:

```bash
# Latest version
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash

# Specific version
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash -s -- v0.2.0

# Custom install directory
INSTALL_DIR=~/.local/bin curl -sSL .../install.sh | bash
```

### scripts/release.sh

The main release script. Calls `build.sh` internally, plus handles everything else:

```
1. Pre-flight checks (clean tree, tag doesn't exist)
2. Run tests (go test ./...)
3. Update VERSION file
4. Cross-compile 5 platforms (via build.sh --all)
5. Generate checksums (sha256)
6. Commit + create git tag
7. Push tag → GoReleaser builds + publishes on GitHub Actions
```

**Default mode** (push tag, let GoReleaser handle the release):
```bash
./scripts/release.sh 0.2.0
```

**Local mode** (build + upload directly via `gh` CLI, no GoReleaser needed):
```bash
./scripts/release.sh 0.2.0 --local
```

---

## GitHub Actions

Three workflows in `.github/workflows/`:

### ci.yml (on push/PR)

Runs on every push to main and every PR:
- Tests on Ubuntu + macOS
- Lint (staticcheck)
- Build smoke test

### release.yml (on tag push)

Triggered when you push a `v*` tag:
- Extracts release notes from CHANGELOG.md
- GoReleaser runs tests, builds binaries for all 5 platform targets
- Creates GitHub Release with CHANGELOG notes
- Uploads all binaries as release assets

**This is what `scripts/release.sh` triggers** — you push the tag, GitHub Actions does the rest.

### deploy-docs.yml (on push to main)

Builds and deploys the documentation site via MkDocs Material to GitHub Pages.

---

## Installation Methods (for users)

### Go install

```bash
go install github.com/aiperceivable/unirelease@latest
```

### One-line installer

```bash
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash
```

### Download binary

From [GitHub Releases](https://github.com/aiperceivable/unirelease/releases):

```bash
# macOS (Apple Silicon)
curl -Lo unirelease https://github.com/aiperceivable/unirelease/releases/latest/download/unirelease-darwin-arm64
chmod +x unirelease && sudo mv unirelease /usr/local/bin/

# Linux
curl -Lo unirelease https://github.com/aiperceivable/unirelease/releases/latest/download/unirelease-linux-amd64
chmod +x unirelease && sudo mv unirelease /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/aiperceivable/unirelease.git
cd unirelease && go build -o unirelease . && ./unirelease --help
```

---

## Release Checklist

When releasing a new version of unirelease:

1. Update `CHANGELOG.md` with the new version section
2. Run: `./scripts/release.sh X.Y.Z`
3. Verify: https://github.com/aiperceivable/unirelease/releases

That's the whole process. The script handles everything else.
