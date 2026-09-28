#!/bin/sh
set -e

# Hikari installer script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.sh | sh

REPO="NubiMa/hikari-cli"
BINARY="hikari"
MODULE="github.com/NubiMa/hikari-cli"

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
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7*|armhf)  ARCH="armv7" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected platform: ${OS}_${ARCH}"

# Determine install directory (no sudo needed for ~/.local/bin)
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -n "$HOME" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
else
  INSTALL_DIR="/usr/local/bin"
fi

# Fetch latest release version from GitHub API
LATEST_TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name":' \
  | sed -E 's/.*"([^"]+)".*/\1/' \
  || true)

# ── Fallback: build from source ─────────────────────────────────────────────
build_from_source() {
  if command -v go > /dev/null 2>&1; then
    echo "Building from source with Go..."
    # CGO_ENABLED=0 produces a fully static binary (no libc dependency)
    CGO_ENABLED=0 go install "${MODULE}/cmd/hikari@latest"
    GOPATH_BIN="$(go env GOPATH)/bin"
    echo ""
    echo "✓ Hikari installed to ${GOPATH_BIN}/hikari"
    echo "  Make sure ${GOPATH_BIN} is in your PATH:"
    echo "    export PATH=\"\$PATH:${GOPATH_BIN}\""
  else
    echo ""
    echo "Error: Could not fetch a prebuilt binary and 'go' is not installed."
    echo ""
    echo "Options:"
    echo "  1. Install Go (https://go.dev/dl/) then re-run this script."
    echo "  2. Clone the repo and run: make build"
    echo "     git clone https://github.com/${REPO}.git && cd hikari-cli && make build"
    exit 1
  fi
}

if [ -z "$LATEST_TAG" ]; then
  echo "Could not fetch latest release tag (no releases published yet or no network access)."
  build_from_source
  exit 0
fi

VERSION="${LATEST_TAG#v}"
TARBALL="hikari_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

echo "Downloading Hikari ${LATEST_TAG} from ${URL}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

if curl -fsSL "$URL" -o "$TMP_DIR/$TARBALL" 2>/dev/null; then
  tar -xzf "$TMP_DIR/$TARBALL" -C "$TMP_DIR"
  install -m 755 "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
  echo ""
  echo "✓ Hikari ${LATEST_TAG} installed to $INSTALL_DIR/$BINARY"
  echo ""
  echo "Run 'hikari --init' to generate your initial config, then start with 'hikari'."
else
  echo "Prebuilt release binary not found for ${OS}_${ARCH}. Falling back to source build."
  build_from_source
fi
