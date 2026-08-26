#!/bin/sh
set -e

echo "==> Welcome to the Bit package manager (bitpkg) installer"
if [ "$(id -u)" -ne 0 ]; then
    echo "Please run this installer as root (sudo or doas)."
    exit 1
fi

REAL_USER=${SUDO_USER:-${DOAS_USER:-root}}
if [ "$REAL_USER" = "root" ]; then
    USER_HOME="/root"
else
    USER_HOME="/home/$REAL_USER"
fi

CONFIG_DIR="$USER_HOME/.config/bit"
DATA_DIR="$USER_HOME/.local/share/bit"

echo "==> Compiling bit..."
go build -o /usr/bin/bit main.go
chmod 755 /usr/bin/bit

echo "==> Configuring bit..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"

read -p "Install prefix directory [/usr]: " PREFIX
PREFIX=${PREFIX:-/usr}

read -p "Compiler command (gcc, clang, tcc) [gcc]: " CC
CC=${CC:-gcc}

read -p "Default package mode (binary, source, ask) [ask]: " PKG_MODE
PKG_MODE=${PKG_MODE:-ask}

NPROC=$(nproc 2>/dev/null || echo 1)

cat <<EOF > "$CONFIG_DIR/config"
PREFIX="$PREFIX"
BIT_HOME="$DATA_DIR"
BIT_REPO="https://github.com/apiwo/bit.git"
NPROC="$NPROC"
CC="$CC"
PKG_MODE="$PKG_MODE"
EOF

chown -R "$REAL_USER:$REAL_USER" "$CONFIG_DIR"
chown -R "$REAL_USER:$REAL_USER" "$DATA_DIR"

echo "==> Bit installed successfully! Run 'bit h' for help."
