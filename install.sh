#!/usr/bin/env bash
set -e

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: Root permissions needed." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==> Compiling pm with Go..."
GO111MODULE=off go build -o pm main.go

./pm c

echo ""
echo "==> Installing binary to /usr/bin/pm..."
cp pm /usr/bin/pm
chmod +x /usr/bin/pm

echo "==> Syncing package repository..."
/usr/bin/pm s

echo ""
echo "Setup complete! Run 'pm' for available options."
