#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run initial interactive configuration
"$SCRIPT_DIR/pm" c

echo ""
echo "==> Installing binary to /usr/bin/pm..."

if [ "$(id -u)" -ne 0 ]; then
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

echo "==> Syncing package repository..."
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
echo "Setup complete! Run 'pm' for available options."
