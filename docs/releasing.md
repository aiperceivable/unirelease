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
| `scripts/build.sh` | Cross-compile all platforms | `./scripts/build.sh [version]` |
| `scripts/install.sh` | One-line installer for users | `curl -sSL .../install.sh \| bash` |
| `scripts/release.sh` | Full release pipeline | `./scripts/release.sh <version> [--local]` |

### scripts/build.sh

Cross-compiles for all platforms without releasing:

```bash
./scripts/build.sh         # reads version from VERSION file
./scripts/build.sh 1.0.0   # explicit version

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
curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash -s -- v0.1.0

# Custom install directory
INSTALL_DIR=~/.local/bin curl -sSL .../install.sh | bash
```

### scripts/release.sh

The main release script. Calls `build.sh` internally, plus handles everything else:

```
1. Pre-flight checks (clean tree, tag doesn't exist)
2. Run tests (go test ./...)
3. Update VERSION file
4. Cross-compile 5 platforms (via build.sh)
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

Two workflows in `.github/workflows/`:

### ci.yml (on push/PR)

Runs on every push to main and every PR:
- Tests on Ubuntu + macOS
- Lint (go vet + staticcheck)
- Build smoke test

### release.yml (on tag push)

Triggered when you push a `v*` tag:
- Runs full test suite
- GoReleaser builds binaries for all 5 platform targets
- Creates GitHub Release with CHANGELOG notes
- Uploads all binaries as release assets

**This is what `scripts/release.sh` triggers** — you push the tag, GitHub Actions does the rest.

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

### Docker

```bash
docker build -t unirelease .
docker run --rm -v "$(pwd):/project" -w /project unirelease --dry-run
docker run --rm -e GITHUB_TOKEN="$TOKEN" -v "$(pwd):/project" -w /project unirelease --yes
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
