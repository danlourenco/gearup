#!/usr/bin/env bash
# Install gearup: detect arch, download the latest Bun-compiled binary from
# GitHub Releases, verify checksum, place at $GEARUP_INSTALL_DIR (default
# ~/.local/bin/).
set -euo pipefail

REPO="${GEARUP_REPO:-danlourenco/gearup}"
INSTALL_DIR="${GEARUP_INSTALL_DIR:-$HOME/.local/bin}"

# Detect platform.
OS=""
case "$(uname -s)" in
  Darwin) OS="darwin" ;;
  *) echo "error: gearup currently supports macOS only (Darwin). Detected: $(uname -s)" >&2; exit 1 ;;
esac

ARCH=""
case "$(uname -m)" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64) ARCH="x64" ;;
  *) echo "error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Resolve the latest release tag.
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG="$(curl -fsSL "$LATEST_URL" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
if [ -z "$TAG" ]; then
  echo "error: could not resolve latest release tag from $LATEST_URL" >&2
  exit 1
fi

ARTIFACT_BASE="https://github.com/${REPO}/releases/download/${TAG}"

# Try platform-prefixed name first (gearup-darwin-arm64), fall back to flat (gearup).
BINARY_URL="${ARTIFACT_BASE}/gearup-${OS}-${ARCH}"
SHA_URL="${BINARY_URL}.sha256"
if ! curl -fsSL --head -o /dev/null "$BINARY_URL" 2>/dev/null; then
  BINARY_URL="${ARTIFACT_BASE}/gearup"
  SHA_URL="${BINARY_URL}.sha256"
fi

mkdir -p "$INSTALL_DIR"
TARGET="$INSTALL_DIR/gearup"
TMP="$(mktemp)"
TMP_SHA="$(mktemp)"

cleanup() { rm -f "$TMP" "$TMP_SHA"; }
trap cleanup EXIT

echo "Downloading gearup ${TAG} for ${OS}-${ARCH}..."
curl -fsSL "$BINARY_URL" -o "$TMP"

# Optional checksum verification (only if .sha256 exists).
if curl -fsSL "$SHA_URL" -o "$TMP_SHA" 2>/dev/null; then
  ACTUAL="$(shasum -a 256 "$TMP" | awk '{print $1}')"
  EXPECTED="$(awk '{print $1}' "$TMP_SHA")"
  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "error: checksum mismatch (expected $EXPECTED, got $ACTUAL)" >&2
    exit 1
  fi
  echo "Checksum verified."
fi

chmod +x "$TMP"
mv "$TMP" "$TARGET"

echo
echo "Installed gearup to $TARGET"
"$TARGET" version

cat <<EOF

Add $INSTALL_DIR to your PATH if it isn't already:

  export PATH="$INSTALL_DIR:\$PATH"

Run \`gearup --help\` to get started.
EOF
