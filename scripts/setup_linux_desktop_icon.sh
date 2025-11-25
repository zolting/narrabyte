#!/bin/bash

# Usage: ./scripts/setup_linux_desktop_icon.sh

set -e

# Get the absolute path to the project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

APP_NAME="narrabyte"
ICON_SOURCE="$PROJECT_ROOT/build/appicon.png"
BINARY_PATH="$PROJECT_ROOT/build/bin/narrabyte"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    echo "Please run 'wails build' first."
    exit 1
fi

# Destination paths
DESKTOP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor/1024x1024/apps"

mkdir -p "$DESKTOP_DIR"
mkdir -p "$ICON_DIR"

# Copy icon
cp "$ICON_SOURCE" "$ICON_DIR/$APP_NAME.png"
echo "Installed icon to $ICON_DIR/$APP_NAME.png"

# Create .desktop file
cat > "$DESKTOP_DIR/$APP_NAME.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Narrabyte
Comment=Narrabyte Application
Exec="$BINARY_PATH"
Icon=$APP_NAME
Terminal=false
Categories=Development;
StartupWMClass=narrabyte
EOF

echo "Created desktop entry at $DESKTOP_DIR/$APP_NAME.desktop"

# Update desktop database if the command exists
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database "$DESKTOP_DIR"
fi

echo "Done! You should now see the Narrabyte icon in your application launcher."
