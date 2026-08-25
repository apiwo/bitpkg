#!/usr/bin/env bash
set -e

echo "============================================="
echo "        pm Package Manager Setup             "
echo "============================================="
echo ""

# Quick Configuration Prompts
read -p "Install prefix directory [/usr]: " input_prefix
PREFIX="${input_prefix:-/usr}"

read -p "Package data directory [${HOME}/.local/share/pm]: " input_home
PM_HOME="${input_home:-${HOME}/.local/share/pm}"

read -p "Package repository Git URL [https://github.com/apiwo/pm.git]: " input_repo
PM_REPO="${input_repo:-https://github.com/apiwo/pm.git}"

# Save configuration
CONFIG_DIR="${HOME}/.config/pm"
CONFIG_FILE="${CONFIG_DIR}/config"

mkdir -p "$CONFIG_DIR"
cat << EOF > "$CONFIG_FILE"
# pm configuration
PREFIX="$PREFIX"
PM_HOME="$PM_HOME"
PM_REPO="$PM_REPO"
EOF

echo ""
echo "==> Configuration saved to $CONFIG_FILE"

# Install binary to system
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$(id -u)" -ne 0 ]; then
    echo "==> Elevating permissions to install pm to /usr/bin/pm..."
    if command -v doas >/dev/null 2>&1; then
        doas cp "$SCRIPT_DIR/pm" /usr/bin/pm
        doas chmod +x /usr/bin/pm
    elif command -v sudo >/dev/null 2>&1; then
        sudo cp "$SCRIPT_DIR/pm" /usr/bin/pm
        sudo chmod +x /usr/bin/pm
    else
        echo "Error: Root permissions needed." >&2
        exit 1
    fi
else
    cp "$SCRIPT_DIR/pm" /usr/bin/pm
    chmod +x /usr/bin/pm
fi

echo "==> 'pm' installed to /usr/bin/pm"
echo ""

# Initial sync
if [ "$(id -u)" -ne 0 ]; then
    if command -v doas >/dev/null 2>&1; then
        doas /usr/bin/pm s
    elif command -v sudo >/dev/null 2>&1; then
        sudo /usr/bin/pm s
    fi
else
    /usr/bin/pm s
fi

echo ""
echo "Setup complete! Run 'pm' to see available commands."
