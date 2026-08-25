#!/usr/bin/env bash
set -e

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: Root permissions needed. Run installer with root (e.g. doas ./install.sh or sudo ./install.sh)." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "==> Compiling pm with Go..."


MOD_CREATED=false
if [ ! -f "go.mod" ]; then
    go mod init pm >/dev/null 2>&1 || true
    MOD_CREATED=true
fi

go build -o pm main.go


if [ "$MOD_CREATED" = true ]; then
    rm -f go.mod
fi

echo "==> Running initial configuration..."
./pm c

echo ""
echo "==> Installing binary to /usr/bin/pm..."
cp pm /usr/bin/pm
chmod +x /usr/bin/pm

echo "==> Syncing package repository..."
/usr/bin/pm s

echo ""
echo "Setup complete! Run 'pm' for available options."
