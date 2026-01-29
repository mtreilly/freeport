#!/bin/bash
set -e

# Build
echo "Building fp..."
go build -o fp .

# Install to ~/go/bin (usually in PATH for Go users)
INSTALL_DIR="${GOBIN:-$HOME/go/bin}"
mkdir -p "$INSTALL_DIR"

echo "Installing to $INSTALL_DIR/fp..."
mv fp "$INSTALL_DIR/fp"

echo "Done. Run 'fp --help' to get started."
