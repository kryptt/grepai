#!/bin/sh
set -e

REPO="${GREPAI_REPO:-kryptt/grepai}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    mingw*|msys*|cygwin*) OS="windows" ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Installing grepai $VERSION for $OS/$ARCH..."

# Set extension
EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    EXT="zip"
fi

# Download
FILENAME="grepai_${VERSION#v}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

TMPDIR=$(mktemp -d)
cd "$TMPDIR"

echo "Downloading $URL..."
curl -sSL -o "$FILENAME" "$URL"

# Extract
if [ "$EXT" = "zip" ]; then
    unzip -q "$FILENAME"
else
    tar -xzf "$FILENAME"
fi

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv grepai "$INSTALL_DIR/"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv grepai "$INSTALL_DIR/"
fi

# Cleanup
cd - > /dev/null
rm -rf "$TMPDIR"

echo "grepai $VERSION installed successfully to $INSTALL_DIR/grepai"
echo ""
echo "Get started:"
echo "  cd your-project"
echo "  grepai init"
echo "  grepai watch"
