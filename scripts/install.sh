#!/bin/bash
set -euo pipefail

# Install unirelease binary for the current platform.
#
# Local install (after ./scripts/build.sh):
#   ./scripts/install.sh
#
# Remote install (for end users):
#   curl -sSL https://raw.githubusercontent.com/aiperceivable/unirelease/main/scripts/install.sh | bash
#   curl -sSL ... | bash -s -- v0.1.0

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*)
    OS="windows"
    ;;
  *)
    echo "Error: unsupported OS: $OS"
    exit 1
    ;;
esac

BINARY="unirelease-${OS}-${ARCH}"
[ "$OS" = "windows" ] && BINARY="${BINARY}.exe"

# Check for local dist/ first (after ./scripts/build.sh)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"
ROOT_DIR=""
[ -n "$SCRIPT_DIR" ] && ROOT_DIR="$(dirname "$SCRIPT_DIR")"

LOCAL_BINARY=""
if [ -n "$ROOT_DIR" ] && [ -f "${ROOT_DIR}/dist/${BINARY}" ]; then
  LOCAL_BINARY="${ROOT_DIR}/dist/${BINARY}"
fi

if [ -n "$LOCAL_BINARY" ]; then
  echo "Installing unirelease from local build..."
  echo "  Source:  ${LOCAL_BINARY}"
  echo "  Target:  ${INSTALL_DIR}/unirelease"
  echo ""

  SRC="$LOCAL_BINARY"
else
  # Remote download
  VERSION="${1:-latest}"
  REPO="aiperceivable/unirelease"

  if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"
  else
    VERSION="${VERSION#v}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY}"
  fi

  echo "Installing unirelease from GitHub..."
  echo "  OS:      ${OS}"
  echo "  Arch:    ${ARCH}"
  echo "  URL:     ${DOWNLOAD_URL}"
  echo "  Target:  ${INSTALL_DIR}/unirelease"
  echo ""

  TMPFILE="$(mktemp)"
  trap 'rm -f "$TMPFILE"' EXIT

  if command -v curl &>/dev/null; then
    curl -fsSL -o "$TMPFILE" "$DOWNLOAD_URL"
  elif command -v wget &>/dev/null; then
    wget -qO "$TMPFILE" "$DOWNLOAD_URL"
  else
    echo "Error: curl or wget required"
    exit 1
  fi

  SRC="$TMPFILE"
fi

# Install
chmod +x "$SRC"
if [ -w "$INSTALL_DIR" ]; then
  cp "$SRC" "${INSTALL_DIR}/unirelease"
else
  echo "  (requires sudo to write to ${INSTALL_DIR})"
  sudo cp "$SRC" "${INSTALL_DIR}/unirelease"
fi

echo "Installed: $("${INSTALL_DIR}/unirelease" --help 2>&1 | head -1 || echo "${INSTALL_DIR}/unirelease")"
