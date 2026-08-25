#!/usr/bin/env bash
set -e

if [ "$(id -u)" -ne 0 ]; then
    echo "Error: Root permissions needed." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run initial interactive configuration setup
"$SCRIPT_DIR/pm" c

echo ""
echo "==> Installing binary to /usr/bin/pm..."
cp "$SCRIPT_DIR/pm" /usr/bin/pm
chmod +x /usr/bin/pm

echo "==> Syncing package repository..."
/usr/bin/pm s

echo ""
echo "Setup complete! Run 'pm' for available options."
