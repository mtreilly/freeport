#!/bin/bash
set -e

BINARY_NAME="freeport"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}==>${NC} $1"; }
warn() { echo -e "${YELLOW}==>${NC} $1"; }
error() { echo -e "${RED}==>${NC} $1"; exit 1; }

# Check Go is installed
if ! command -v go &> /dev/null; then
    error "Go is not installed. Please install Go first: https://go.dev/dl/"
fi

# Build the binary
info "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" .

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    info "Creating $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
fi

# Install the binary
info "Installing to $INSTALL_DIR/$BINARY_NAME..."
mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    warn "$INSTALL_DIR is not in your PATH"
    echo ""
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
fi

info "Done! $BINARY_NAME installed successfully."
echo ""
"$INSTALL_DIR/$BINARY_NAME" --help 2>/dev/null | head -5 || true
