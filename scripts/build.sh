#!/bin/bash
set -euo pipefail

# Cross-compile unirelease for all supported platforms.
# Usage: ./scripts/build.sh [version]
#   version: optional, reads from VERSION file if not provided

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

VERSION="${1:-$(cat VERSION 2>/dev/null || echo "dev")}"
LDFLAGS="-s -w -X main.version=${VERSION}"
DIST_DIR="dist"

TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

echo "==> Building unirelease v${VERSION}"
echo ""

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for target in "${TARGETS[@]}"; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  output="${DIST_DIR}/unirelease-${GOOS}-${GOARCH}"
  [ "$GOOS" = "windows" ] && output="${output}.exe"

  printf "  %-20s" "${GOOS}/${GOARCH}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$output" .
  echo "-> $(basename "$output")"
done

echo ""
echo "==> Built $(ls -1 "$DIST_DIR" | wc -l | tr -d ' ') binaries in ${DIST_DIR}/"
ls -lh "$DIST_DIR"/
