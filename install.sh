#!/bin/bash
set -euo pipefail

REPO="danlourenco/gearup"
INSTALL_DIR="${GEARUP_INSTALL_DIR:-$HOME/.local/bin}"

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64)        ARCH="amd64" ;;
  *)
    echo "error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [ "$OS" != "darwin" ]; then
  echo "error: gearup currently supports macOS only (detected: $OS)" >&2
  exit 1
fi

ASSET="gearup_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "Installing gearup..."
echo "  arch: ${ARCH}"
echo "  dest: ${INSTALL_DIR}"

mkdir -p "$INSTALL_DIR"

if ! curl -fsSL "$URL" | tar xz -C "$INSTALL_DIR"; then
  echo "error: download failed. Check https://github.com/${REPO}/releases for available assets." >&2
  exit 1
fi

chmod +x "${INSTALL_DIR}/gearup"

# Check if INSTALL_DIR is on PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  SHELL_NAME="$(basename "$SHELL")"
  RC_FILE="$HOME/.zshrc"
  case "$SHELL_NAME" in
    bash) RC_FILE="$HOME/.bashrc" ;;
    zsh)  RC_FILE="$HOME/.zshrc" ;;
  esac

  echo ""
  echo "Add gearup to your PATH by appending this to ${RC_FILE}:"
  echo ""
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
  echo ""
  echo "Then restart your shell or run: source ${RC_FILE}"
else
  echo ""
  echo "gearup installed successfully!"
  echo ""
  "${INSTALL_DIR}/gearup" version
fi
