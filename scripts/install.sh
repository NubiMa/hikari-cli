#!/bin/sh
set -e

# Hikari installer script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.sh | sh

REPO="NubiMa/hikari-cli"
BINARY="hikari"
MODULE="github.com/NubiMa/hikari-cli"

# Minimum Go version required to build Hikari (matches go.mod)
GO_MIN="1.24.2"
# Latest stable Go to install if none found (updated periodically)
GO_INSTALL_VERSION="1.24.2"

echo "=== Hikari Installer ==="

# ── Platform detection ────────────────────────────────────────────────────────

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

# Determine install directory (prefer /usr/local/bin, fall back to ~/.local/bin)
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -n "$HOME" ]; then
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
else
  INSTALL_DIR="/usr/local/bin"
fi

# ── Go toolchain bootstrap ────────────────────────────────────────────────────

# Returns 0 if version $1 >= version $2, 1 otherwise.
# Compares only the first three numeric parts (x.y.z).
version_ge() {
  # Strip leading 'go' if present
  v1=$(echo "$1" | sed 's/^go//' | tr -d '[:space:]')
  v2=$(echo "$2" | sed 's/^go//' | tr -d '[:space:]')
  # Use awk for portable numeric comparison
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

# install_go installs Go to $HOME/.local/go and prepends it to PATH.
install_go() {
  if [ "$OS" = "windows" ]; then
    echo ""
    echo "Automatic Go installation is not supported on Windows."
    echo "Please install Go manually from https://go.dev/dl/ then re-run this script."
    exit 1
  fi

  GO_INSTALL_DIR="$HOME/.local/go"
  GO_ARCHIVE="go${GO_INSTALL_VERSION}.${OS}-${ARCH}.tar.gz"
  GO_URL="https://go.dev/dl/${GO_ARCHIVE}"

  echo ""
  echo "Go ${GO_MIN}+ is required but not found (or is outdated)."
  echo "Installing Go ${GO_INSTALL_VERSION} to ${GO_INSTALL_DIR}..."
  echo ""

  TMP_GO=$(mktemp -d)
  trap 'rm -rf "$TMP_GO"' EXIT

  if ! curl -fsSL "$GO_URL" -o "$TMP_GO/$GO_ARCHIVE"; then
    echo "Error: failed to download Go from $GO_URL"
    echo "Please install Go manually: https://go.dev/dl/"
    exit 1
  fi

  # Remove any previous local installation
  rm -rf "$GO_INSTALL_DIR"
  mkdir -p "$HOME/.local"
  tar -xzf "$TMP_GO/$GO_ARCHIVE" -C "$HOME/.local"
  # tar extracts to $HOME/.local/go

  echo "✓ Go ${GO_INSTALL_VERSION} installed to ${GO_INSTALL_DIR}"

  # Prepend to PATH for the rest of this script
  export PATH="$GO_INSTALL_DIR/bin:$PATH"

  # Provide a shell hint
  echo ""
  echo "  To make this permanent, add the following to your shell profile"
  echo "  (~/.bashrc, ~/.zshrc, ~/.profile, etc.):"
  echo ""
  echo "    export PATH=\"\$HOME/.local/go/bin:\$PATH\""
  echo ""
}

# Ensure a compatible Go is available.
# If run from a cloned repository, delegates to bootstrap_go.sh.
# Otherwise (curl | sh) uses the inline install_go function.
ensure_go() {
  SCRIPT_DIR="$(cd "$(dirname "$0")" 2>/dev/null && pwd || echo "")"
  BOOTSTRAP="$SCRIPT_DIR/bootstrap_go.sh"

  if [ -f "$BOOTSTRAP" ]; then
    # Running from the cloned repo — use the shared script
    sh "$BOOTSTRAP"
    # bootstrap_go.sh installs to $HOME/.local/go; prepend for this shell
    export PATH="$HOME/.local/go/bin:$PATH"
    return 0
  fi

  # Running via curl | sh — use inline logic
  if command -v go > /dev/null 2>&1; then
    CURRENT_GO_VERSION="$(go version | awk '{print $3}' | sed 's/^go//')"
    if version_ge "$CURRENT_GO_VERSION" "$GO_MIN"; then
      return 0  # Already have a sufficient Go
    fi
    echo "Installed Go ${CURRENT_GO_VERSION} is older than required ${GO_MIN}."
  fi
  install_go
}

# ── Build from source ─────────────────────────────────────────────────────────

build_from_source() {
  ensure_go

  echo "Building Hikari from source with Go $(go version | awk '{print $3}')..."
  CGO_ENABLED=0 go install "${MODULE}/cmd/hikari@latest"
  GOPATH_BIN="$(go env GOPATH)/bin"
  echo ""
  echo "✓ Hikari installed to ${GOPATH_BIN}/hikari"
  echo ""
  echo "  Make sure ${GOPATH_BIN} is in your PATH:"
  echo "    export PATH=\"\$PATH:${GOPATH_BIN}\""
  echo ""
  echo "Run 'hikari --init' to generate your initial config, then start with 'hikari'."
}

# ── Fetch latest release version ──────────────────────────────────────────────

LATEST_TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name":' \
  | sed -E 's/.*"([^"]+)".*/\1/' \
  || true)

if [ -z "$LATEST_TAG" ]; then
  echo "Could not fetch latest release tag (no releases published yet or no network access)."
  build_from_source
  exit 0
fi

# ── Download prebuilt release binary ─────────────────────────────────────────

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
  echo "Prebuilt release binary not found for ${OS}_${ARCH}. Falling back to source build..."
  build_from_source
fi
