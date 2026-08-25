#!/usr/bin/env bash
set -e

PM_REPO="https://github.com/apiwo/pm.git"
PM_HOME="${HOME}/.local/share/pm"
REPO_DIR="$PM_HOME/repo"
BUILD_DIR="$PM_HOME/build"

mkdir -p "$PM_HOME"

run_root() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    elif [ -n "$ROOT_CMD" ]; then
        $ROOT_CMD "$@"
    elif command -v doas >/dev/null 2>&1; then
        doas "$@"
    elif command -v sudo >/dev/null 2>&1; then
        sudo "$@"
    elif command -v pkexec >/dev/null 2>&1; then
        pkexec "$@"
    elif command -v su >/dev/null 2>&1; then
        su -c "$*"
    else
        echo "Error: No root privilege tool found and you are not root." >&2
        exit 1
    fi
}

usage() {
    echo "pm - A simple source-based package manager"
    echo "Usage:"
    echo "  pm s         - Sync / update local repository from GitHub"
    echo "  pm u         - Alias for sync (pm s)"
    echo "  pm b <pkg>   - Build package recipe (as user)"
    echo "  pm bi <pkg>  - Build and install package recipe"
    exit 1
}

pm_sync() {
    echo "==> Syncing packages from $PM_REPO..."
    if [ -d "$REPO_DIR/.git" ]; then
        git -C "$REPO_DIR" pull
    else
        git clone "$PM_REPO" "$REPO_DIR"
    fi
    echo "==> Sync complete."
}

pm_build() {
    local pkg="$1"
    if [ -z "$pkg" ]; then
        echo "Error: No package specified."
        exit 1
    fi

    local recipe_dir="$REPO_DIR/main/$pkg"
    if [ ! -f "$recipe_dir/recipe" ]; then
        echo "Error: Package '$pkg' not found in main/ directory!"
        exit 1
    fi

    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"

    echo "==> Loading recipe for $pkg..."
    source "$recipe_dir/recipe"

    echo "==> Downloading source: $SRC_URL..."
    wget -q --show-progress "$SRC_URL" -O source_archive
    
    rm -rf src_extract
    mkdir src_extract
    tar -xf source_archive -C src_extract 2>/dev/null || unzip -q source_archive -d src_extract 2>/dev/null || tar -xJf source_archive -C src_extract 2>/dev/null || true
    
    cd src_extract
    cd "$(ls -d */ | head -n 1)"

    echo "==> Building $pkg..."
    if [ -n "$BUILD_CMD" ]; then
        eval "$BUILD_CMD"
    else
        ./configure --prefix=/usr
        make -j$(nproc)
    fi
}

pm_build_install() {
    local pkg="$1"
    pm_build "$pkg"
    
    local recipe_dir="$REPO_DIR/main/$pkg"
    cd "$BUILD_DIR/src_extract/$(ls -t "$BUILD_DIR/src_extract" | head -n 1)"

    echo "==> Installing $pkg..."
    source "$recipe_dir/recipe"

    if [ -n "$INSTALL_CMD" ]; then
        local clean_install_cmd="${INSTALL_CMD#sudo }"
        clean_install_cmd="${clean_install_cmd#doas }"
        clean_install_cmd="${clean_install_cmd#pkexec }"
        run_root eval "$clean_install_cmd"
    else
        run_root make install
    fi
    echo "==> Successfully installed $pkg!"
}

case "$1" in
    s|u)
        pm_sync
        ;;
    b)
        pm_build "$2"
        ;;
    bi)
        pm_build_install "$2"
        ;;
    *)
        usage
        ;;
esac
