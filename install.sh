#!/bin/bash
# AutoEngineer installer

set -e

VERSION="2.4.1"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
GITHUB_REPO="liam-witterick/autoengineer"
BINARY_NAME="autoengineer"

echo ""
echo "📦 Installing AutoEngineer v$VERSION"
echo "===================================="
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux)  PLATFORM="linux" ;;
  darwin) PLATFORM="darwin" ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

BINARY="${BINARY_NAME}-${PLATFORM}-${ARCH}"
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${BINARY}"

echo "🔍 Detected: ${PLATFORM}/${ARCH}"
echo ""

# Create install directory if needed
mkdir -p "$INSTALL_DIR"

# Download the binary
echo "⬇️  Downloading ${BINARY}..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"; then
  echo ""
  echo "❌ Failed to download from: $DOWNLOAD_URL"
  echo ""
  echo "The release v${VERSION} may not exist yet."
  echo "Check available releases at: https://github.com/${GITHUB_REPO}/releases"
  exit 1
fi
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo ""
  echo "⚠️  $INSTALL_DIR is not in your PATH"
  echo ""
  echo "Add this to your ~/.bashrc or ~/.zshrc:"
  echo ""
  echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
  echo ""
fi

# Verify dependencies
echo ""
echo "🔍 Checking dependencies..."

MISSING=""
for cmd in copilot gh jq; do
  if command -v $cmd &> /dev/null; then
    echo "   ✅ $cmd"
  else
    echo "   ❌ $cmd (missing)"
    MISSING="$MISSING $cmd"
  fi
done

if [[ -n "$MISSING" ]]; then
  echo ""
  echo "⚠️  Missing dependencies:$MISSING"
  echo ""
  echo "Install them:"
  echo "   copilot: https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line"
  echo "   gh:      brew install gh  OR  https://cli.github.com/"
  echo "   jq:      brew install jq  OR  apt install jq"
fi

echo ""
echo "✅ Installed to: $INSTALL_DIR/$BINARY_NAME"
echo ""
echo "Run '$BINARY_NAME --help' to get started"
echo ""