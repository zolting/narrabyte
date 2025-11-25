#!/bin/bash
# Narrabyte Linux Installer (for release packages)

set -e

# Directory where this script is running from (the extracted folder)
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

APP_NAME="narrabyte"
BINARY_NAME="narrabyte"
ICON_NAME="appicon.png"

# Target user directories
INSTALL_BIN_DIR="$HOME/.local/bin"
INSTALL_ICON_DIR="$HOME/.local/share/icons/hicolor/1024x1024/apps"
INSTALL_DESKTOP_DIR="$HOME/.local/share/applications"

# Ensure directories exist
mkdir -p "$INSTALL_BIN_DIR"
mkdir -p "$INSTALL_ICON_DIR"
mkdir -p "$INSTALL_DESKTOP_DIR"

# 1. Install Binary
echo "Installing binary to $INSTALL_BIN_DIR..."
if [ -f "$SOURCE_DIR/$BINARY_NAME" ]; then
    cp "$SOURCE_DIR/$BINARY_NAME" "$INSTALL_BIN_DIR/$APP_NAME"
    chmod +x "$INSTALL_BIN_DIR/$APP_NAME"
else
    echo "Error: Binary '$BINARY_NAME' not found in $SOURCE_DIR"
    exit 1
fi

# 2. Install Icon
echo "Installing icon to $INSTALL_ICON_DIR..."
if [ -f "$SOURCE_DIR/$ICON_NAME" ]; then
    cp "$SOURCE_DIR/$ICON_NAME" "$INSTALL_ICON_DIR/$APP_NAME.png"
else
    echo "Warning: Icon '$ICON_NAME' not found in $SOURCE_DIR. Desktop entry might miss icon."
fi

# 3. Create Desktop Entry
echo "Creating desktop entry in $INSTALL_DESKTOP_DIR..."
cat > "$INSTALL_DESKTOP_DIR/$APP_NAME.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Narrabyte
Comment=Narrabyte Application
Exec="$INSTALL_BIN_DIR/$APP_NAME"
Icon=$APP_NAME
Terminal=false
Categories=Development;
StartupWMClass=narrabyte
EOF

# 4. Refresh desktop database
if command -v update-desktop-database &> /dev/null; then
    update-desktop-database "$INSTALL_DESKTOP_DIR"
fi

echo "-------------------------------------------------------"
echo "Installation Complete!"
echo "1. You can run the app by typing '$APP_NAME' (if $INSTALL_BIN_DIR is in your PATH)."
echo "2. You should see 'Narrabyte' in your application launcher."
echo "-------------------------------------------------------"
