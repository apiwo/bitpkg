#!/usr/bin/env bash
set -e

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: Root permissions needed. Run installer with root (e.g. doas ./install.sh or sudo ./install.sh)." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Compiling pm with Go..."
go build -o "$SCRIPT_DIR/pm" "$SCRIPT_DIR/main.go"

"$SCRIPT_DIR/pm" c

echo ""
echo "==> Installing binary to /usr/bin/pm..."
cp "$SCRIPT_DIR/pm" /usr/bin/pm
chmod +x /usr/bin/pm

echo "==> Syncing package repository..."
/usr/bin/pm s

echo ""
echo "Setup complete! Run 'pm' for available options."
