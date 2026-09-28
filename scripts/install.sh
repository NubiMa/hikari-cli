#!/bin/sh
set -e

# Hikari installer script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/nubiv/hikari/main/scripts/install.sh | sh

REPO="nubiv/hikari"
BINARY="hikari"

echo "=== Hikari Installer ==="

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  msys*|mingw*|cygwin*) OS="windows" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7*|armhf) ARCH="armv7" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected platform: ${OS}_${ARCH}"

# Determine install directory
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -n "$HOME" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
else
  INSTALL_DIR="/usr/bin"
fi

# Fetch latest release version
LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)

if [ -z "$LATEST_TAG" ]; then
  echo "Could not fetch latest release tag via GitHub API, falling back to git or local build."
  if command -v go >/dev/null 2>&1; then
    echo "Building from source with Go..."
    go install "github.com/${REPO}/cmd/hikari@latest"
    echo "Hikari installed successfully via 'go install' to $(go env GOPATH)/bin/hikari"
    exit 0
  else
    echo "Error: Neither prebuilt binary nor Go compiler found."
    exit 1
  fi
fi

VERSION="${LATEST_TAG#v}"
TARBALL="hikari_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

echo "Downloading Hikari ${LATEST_TAG} from ${URL}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if curl -sL --fail "$URL" -o "$TMP_DIR/$TARBALL"; then
  tar -xzf "$TMP_DIR/$TARBALL" -C "$TMP_DIR"
  mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
  echo "✓ Hikari installed to $INSTALL_DIR/$BINARY"
  echo ""
  echo "Run 'hikari --init' to generate your initial config, then start with 'hikari'."
else
  echo "Prebuilt release binary not found for ${OS}_${ARCH}."
  if command -v go >/dev/null 2>&1; then
    echo "Building from source with Go..."
    go install "github.com/${REPO}/cmd/hikari@latest"
    echo "✓ Hikari installed successfully via 'go install'"
  else
    echo "Please build from source using 'make build'."
    exit 1
  fi
fi
