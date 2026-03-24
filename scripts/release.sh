#!/bin/bash
set -euo pipefail

# Full release pipeline for unirelease itself.
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh 0.2.0
#
# What it does:
#   1. Validates version format
#   2. Runs tests
#   3. Updates VERSION file
#   4. Cross-compiles all platforms
#   5. Creates checksums
#   6. Creates git tag
#   7. Pushes tag (triggers GitHub Actions → GoReleaser)
#
# For local-only release (no GoReleaser), add --local flag:
#   ./scripts/release.sh 0.2.0 --local
#   This creates a GitHub Release directly via gh CLI with local binaries.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

# --- Parse args ---
VERSION="${1:-}"
LOCAL_MODE=false
if [ "${2:-}" = "--local" ]; then
  LOCAL_MODE=true
fi

if [ -z "$VERSION" ]; then
  echo "Usage: ./scripts/release.sh <version> [--local]"
  echo "Example: ./scripts/release.sh 0.2.0"
  exit 1
fi

if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: version must be semver (X.Y.Z), got: $VERSION"
  exit 1
fi

TAG="v${VERSION}"

echo "==> Releasing unirelease ${TAG}"
echo ""

# --- Step 1: Pre-flight checks ---
echo "[1/7] Pre-flight checks..."

if ! git diff-index --quiet HEAD -- 2>/dev/null; then
  echo "  Warning: uncommitted changes detected"
  read -p "  Continue anyway? [y/N] " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
  fi
fi

if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "  Error: tag $TAG already exists locally"
  exit 1
fi

echo "  OK"

# --- Step 2: Run tests ---
echo "[2/7] Running tests..."
go test ./... -count=1 -timeout 5m
echo "  All tests passed"

# --- Step 3: Update VERSION ---
echo "[3/7] Updating VERSION file..."
echo "$VERSION" > VERSION
echo "  VERSION -> $VERSION"

# --- Step 4: Build all platforms ---
echo "[4/7] Cross-compiling..."
bash "$SCRIPT_DIR/build.sh" --all "$VERSION"

# --- Step 5: Checksums ---
echo "[5/7] Generating checksums..."
cd dist
shasum -a 256 unirelease-* > checksums.txt
cat checksums.txt
cd "$ROOT_DIR"

# --- Step 6: Commit + Tag ---
echo "[6/7] Creating git tag..."
git add VERSION
git commit -m "Release ${TAG}" --allow-empty
git tag -a "$TAG" -m "Release ${TAG}"
echo "  Created tag $TAG"

# --- Step 7: Push or local release ---
if [ "$LOCAL_MODE" = true ]; then
  echo "[7/7] Creating local GitHub Release..."

  if ! command -v gh &>/dev/null; then
    echo "  Error: gh CLI required for --local mode"
    echo "  Install: https://cli.github.com/"
    exit 1
  fi

  # Extract release notes from CHANGELOG
  NOTES=$(awk "
    /^## \[${VERSION}\]/ {found=1; next}
    found && /^## \[/ {exit}
    found {print}
  " CHANGELOG.md | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

  if [ -z "$NOTES" ]; then
    NOTES="Release version ${VERSION}"
  fi

  NOTES_FILE=$(mktemp)
  echo "$NOTES" > "$NOTES_FILE"

  git push origin HEAD "$TAG"

  gh release create "$TAG" \
    --title "Release ${VERSION}" \
    --notes-file "$NOTES_FILE" \
    dist/unirelease-*

  rm -f "$NOTES_FILE"
  echo "  Release created with $(ls -1 dist/unirelease-* | wc -l | tr -d ' ') assets"
else
  echo "[7/7] Pushing tag (GoReleaser will handle the release)..."
  git push origin HEAD "$TAG"
  echo "  Pushed. GitHub Actions will build and release."
  echo "  Watch: https://github.com/aiperceivable/unirelease/actions"
fi

echo ""
echo "==> Release ${TAG} complete!"
echo ""
echo "Verify:"
echo "  https://github.com/aiperceivable/unirelease/releases/tag/${TAG}"
