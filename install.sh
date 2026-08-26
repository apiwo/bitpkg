#!/bin/sh
set -e

echo "==> Compiling pm"

SRC_DIR="$(dirname "$(realpath "$0")")"
cd "$SRC_DIR"

GO111MODULE=off go build -o /usr/bin/pm ./main.go

chmod +x /usr/bin/pm
echo "==> Installation complete!"
