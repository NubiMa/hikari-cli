#!/bin/sh
# bootstrap_go.sh — Ensure a compatible Go toolchain is available.
#
# Called by:
#   - Makefile (make bootstrap / make deps)
#   - install.sh (before building from source)
#
# Installs Go ${GO_INSTALL_VERSION} to $HOME/.local/go if Go is not found
# or the installed version is older than GO_MIN_VERSION.
# After running, Go is available at $HOME/.local/go/bin/go.

set -e

GO_MIN_VERSION="${GO_MIN_VERSION:-1.24.2}"
GO_INSTALL_VERSION="${GO_INSTALL_VERSION:-1.24.2}"

# ── Helpers ──────────────────────────────────────────────────────────────────

# version_ge v1 v2 → returns 0 (success) if v1 >= v2
version_ge() {
  v1=$(echo "$1" | sed 's/^go//' | tr -d '[:space:]')
  v2=$(echo "$2" | sed 's/^go//' | tr -d '[:space:]')
  awk -v a="$v1" -v b="$v2" 'BEGIN {
    split(a, va, ".")
    split(b, vb, ".")
    for (i = 1; i <= 3; i++) {
      ai = va[i]+0; bi = vb[i]+0
      if (ai > bi) { exit 0 }
      if (ai < bi) { exit 1 }
    }
    exit 0
  }'
}

# ── Platform detection ────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)  OS="linux" ;;
  darwin*) OS="darwin" ;;
  *)
    echo "bootstrap_go: unsupported OS '$OS'. Install Go manually: https://go.dev/dl/"
    exit 1
    ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7*|armhf)  ARCH="armv7" ;;
  *)
    echo "bootstrap_go: unsupported architecture '$ARCH'. Install Go manually: https://go.dev/dl/"
    exit 1
    ;;
esac

# ── Check existing Go installation ───────────────────────────────────────────

GO_INSTALL_DIR="${GO_INSTALL_DIR:-$HOME/.local/go}"

# Prefer the locally installed Go so we don't accidentally use a system Go
# that is too old.
export PATH="$GO_INSTALL_DIR/bin:$PATH"

if command -v go > /dev/null 2>&1; then
  CURRENT_VERSION="$(go version | awk '{print $3}' | sed 's/^go//')"
  if version_ge "$CURRENT_VERSION" "$GO_MIN_VERSION"; then
    echo "bootstrap_go: Go ${CURRENT_VERSION} is already installed and satisfies >=${GO_MIN_VERSION}. Nothing to do."
    exit 0
  fi
  echo "bootstrap_go: Go ${CURRENT_VERSION} is installed but ${GO_MIN_VERSION}+ is required. Upgrading..."
else
  echo "bootstrap_go: Go not found. Installing Go ${GO_INSTALL_VERSION}..."
fi

# ── Download and install Go ───────────────────────────────────────────────────

GO_ARCHIVE="go${GO_INSTALL_VERSION}.${OS}-${ARCH}.tar.gz"
GO_URL="https://go.dev/dl/${GO_ARCHIVE}"
TMP_DIR=$(mktemp -d)

# Ensure cleanup even if the script is interrupted
trap 'rm -rf "$TMP_DIR"' EXIT

echo "  Downloading ${GO_URL}..."
if ! curl -fsSL --progress-bar "$GO_URL" -o "$TMP_DIR/$GO_ARCHIVE"; then
  echo ""
  echo "Error: failed to download Go from $GO_URL"
  echo "Please install Go manually: https://go.dev/dl/"
  exit 1
fi

echo "  Extracting to ${GO_INSTALL_DIR}..."
rm -rf "$GO_INSTALL_DIR"
mkdir -p "$(dirname "$GO_INSTALL_DIR")"
tar -xzf "$TMP_DIR/$GO_ARCHIVE" -C "$(dirname "$GO_INSTALL_DIR")"
# The archive always unpacks to a directory named 'go', so rename to match our target
EXTRACTED="$(dirname "$GO_INSTALL_DIR")/go"
if [ "$EXTRACTED" != "$GO_INSTALL_DIR" ]; then
  mv "$EXTRACTED" "$GO_INSTALL_DIR"
fi

# Verify
if ! "$GO_INSTALL_DIR/bin/go" version > /dev/null 2>&1; then
  echo "Error: Go installation failed. Binary not found at $GO_INSTALL_DIR/bin/go"
  exit 1
fi

INSTALLED_VERSION="$("$GO_INSTALL_DIR/bin/go" version | awk '{print $3}')"
echo ""
echo "✓ ${INSTALLED_VERSION} installed to ${GO_INSTALL_DIR}"
echo ""
echo "  To make this permanent, add to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
echo ""
echo "    export PATH=\"\$HOME/.local/go/bin:\$PATH\""
echo ""
echo "  For the current session:"
echo "    export PATH=\"${GO_INSTALL_DIR}/bin:\$PATH\""
echo ""
