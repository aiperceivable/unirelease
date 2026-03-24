#!/bin/bash
set -euo pipefail

# Build unirelease binaries.
# Usage:
#   ./scripts/build.sh              Build for current platform only
#   ./scripts/build.sh --all        Cross-compile for all platforms
#   ./scripts/build.sh 1.2.3        Build current platform with version
#   ./scripts/build.sh --all 1.2.3  Cross-compile all with version

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

ALL_TARGETS=false
VERSION=""

for arg in "$@"; do
  case "$arg" in
    --all) ALL_TARGETS=true ;;
    *)     VERSION="$arg" ;;
  esac
done

VERSION="${VERSION:-$(cat VERSION 2>/dev/null || echo "dev")}"
LDFLAGS="-s -w -X main.version=${VERSION}"
DIST_DIR="dist"

ALL_PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

if [ "$ALL_TARGETS" = true ]; then
  echo "==> Building unirelease v${VERSION} (all platforms)"
  echo ""
  TARGETS=("${ALL_PLATFORMS[@]}")
else
  LOCAL_OS="$(go env GOOS)"
  LOCAL_ARCH="$(go env GOARCH)"
  echo "==> Building unirelease v${VERSION} (${LOCAL_OS}/${LOCAL_ARCH})"
  echo ""
  TARGETS=("${LOCAL_OS}/${LOCAL_ARCH}")
fi

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
