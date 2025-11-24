#!/bin/bash

set -e

# portctl installation script
echo "🚀 Installing portctl..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Build the binary
echo "📦 Building portctl for $OS-$ARCH..."
if command -v go &> /dev/null; then
    go build -ldflags "-s -w" -o portctl .
    echo "✅ Built successfully!"
else
    echo "❌ Go is not installed. Please install Go first: https://golang.org/dl/"
    exit 1
fi

# Install the binary
INSTALL_DIR="/usr/local/bin"
if [[ "$OS" == "linux" ]] || [[ "$OS" == "darwin" ]]; then
    if [[ $EUID -eq 0 ]]; then
        # Running as root
        echo "📁 Installing to $INSTALL_DIR..."
        mv portctl "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/portctl"
        echo "✅ portctl installed to $INSTALL_DIR"
    else
        # Not running as root, try with sudo
        echo "📁 Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv portctl "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/portctl"
        echo "✅ portctl installed to $INSTALL_DIR"
    fi
else
    echo "📁 Binary built as 'portctl'. Please move it to your PATH manually."
fi

# Verify installation
if command -v portctl &> /dev/null; then
    echo "🎉 Installation successful!"
    echo ""
    echo "Try it out:"
    echo "  portctl list           # List all processes with open ports"
    echo "  portctl list 8080      # List processes on port 8080"
    echo "  portctl kill 8080      # Kill processes on port 8080"
    echo "  portctl --help         # Show help"
    echo ""
    echo "For more examples, see: https://github.com/ckodex-labs/portctl"
else
    echo "⚠️  Installation may have failed. Please check your PATH."
    echo "Current binary location: $(pwd)/portctl"
fi
