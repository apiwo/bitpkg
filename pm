#!/usr/bin/env bash
set -e

PM_CONFIG="${HOME}/.config/pm/config"

# Load user configuration if present
if [ -f "$PM_CONFIG" ]; then
    source "$PM_CONFIG"
fi

# Fallback defaults if configuration is missing
PM_REPO="${PM_REPO:-https://github.com/apiwo/pm.git}"
PM_HOME="${PM_HOME:-${HOME}/.local/share/pm}"
PREFIX="${PREFIX:-/usr}"
NPROC="${NPROC:-$(nproc 2>/dev/null || echo 1)}"
REPO_DIR="$PM_HOME/repo"
BUILD_DIR="$PM_HOME/build"
INSTALLED_FILE="$PM_HOME/installed"

mkdir -p "$PM_HOME"
touch "$INSTALLED_FILE"

# Standard permission check: Ensure the user running the command has root privileges
require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        echo "Error: Root permissions needed." >&2
        exit 1
    fi
}

usage() {
    echo "pm - A simple source-based package manager"
    echo "Usage:"
    echo "  pm, pm h        - Show this help menu"
    echo "  pm l            - List installed packages"
    echo "  pm c            - Open interactive configuration menu"
    echo "  pm s, pm u      - Sync / update local repository from GitHub (requires root)"
    echo "  pm re           - Reinstall / update 'pm' itself from GitHub (requires root)"
    echo "  pm b [-s] <pkg>  - Build package recipe (requires root, -s to skip prompt)"
    echo "  pm bi [-s] <pkg> - Build and install package recipe (requires root, -s to skip prompt)"
    echo "  pm i [-s] <pkg>  - Install an already compiled package (requires root, -s to skip prompt)"
    echo "  pm r [-s] <pkg>  - Remove package and clean build artifacts (requires root, -s to skip prompt)"
    exit 0
}

pm_config() {
    echo "============================================="
    echo "        pm Configuration Menu                "
    echo "============================================="
    echo ""
    read -p "Install prefix directory [$PREFIX]: " input_prefix
    PREFIX="${input_prefix:-$PREFIX}"

    read -p "Package data directory [$PM_HOME]: " input_home
    PM_HOME="${input_home:-$PM_HOME}"

    read -p "Package repository Git URL [$PM_REPO]: " input_repo
    PM_REPO="${input_repo:-$PM_REPO}"

    local max_cores
    max_cores="$(nproc 2>/dev/null || echo 1)"
    read -p "Number of cores to use for building (1-$max_cores) [$NPROC]: " input_nproc
    NPROC="${input_nproc:-$NPROC}"

    CONFIG_DIR="${HOME}/.config/pm"
    CONFIG_FILE="${CONFIG_DIR}/config"

    mkdir -p "$CONFIG_DIR"
    cat << EOF > "$CONFIG_FILE"
# pm configuration
PREFIX="$PREFIX"
PM_HOME="$PM_HOME"
PM_REPO="$PM_REPO"
NPROC="$NPROC"
EOF

    echo ""
    echo "==> Configuration saved to $CONFIG_FILE"
}

confirm_action() {
    local prompt_msg="$1"
    local skip_flag="$2"
    
    if [ "$skip_flag" = "true" ]; then
        return 0
    fi

    read -p "$prompt_msg [Y/n]: " choice
    case "$choice" in
        [nN][oO]|[nN])
            echo "==> Aborted by user."
            exit 0
            ;;
        *)
            return 0
            ;;
    esac
}

pm_list() {
    echo "==> Installed packages:"
    if [ ! -s "$INSTALLED_FILE" ]; then
        echo "  (No packages installed yet)"
    else
        cat "$INSTALLED_FILE"
    fi
}

pm_sync() {
    require_root
    echo "==> Syncing packages from $PM_REPO..."
    if [ -d "$REPO_DIR/.git" ]; then
        git -C "$REPO_DIR" pull
    else
        git clone "$PM_REPO" "$REPO_DIR"
    fi
    echo "==> Sync complete."
}

pm_reinstall_self() {
    require_root
    echo "==> Updating 'pm' itself from GitHub..."
    local temp_dir
    temp_dir="$(mktemp -d)"
    git clone --quiet "$PM_REPO" "$temp_dir/pm_src"
    
    if [ -f "$temp_dir/pm_src/pm" ]; then
        cp "$temp_dir/pm_src/pm" /usr/bin/pm
        chmod +x /usr/bin/pm
        rm -rf "$temp_dir"
        echo "==> 'pm' has been successfully updated to the latest version!"
        echo "==> Your configuration file at ~/.config/pm/config remains intact."
        exit 0
    else
        echo "Error: Could not find 'pm' script in the repository root." >&2
        rm -rf "$temp_dir"
        exit 1
    fi
}

pm_build() {
    require_root
    local skip_confirm="false"
    if [ "$1" = "-s" ]; then
        skip_confirm="true"
        shift
    fi

    local pkg="$1"
    if [ -z "$pkg" ]; then
        echo "Error: No package specified." >&2
        exit 1
    fi

    local recipe_dir="$REPO_DIR/main/$pkg"
    if [ ! -f "$recipe_dir/recipe" ]; then
        echo "Error: Package '$pkg' not found in main/ directory!" >&2
        exit 1
    fi

    confirm_action "Do you wish to build $pkg?" "$skip_confirm"

    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"

    echo "==> Loading recipe for $pkg..."
    source "$recipe_dir/recipe"

    echo "==> Downloading source: $SRC_URL..."
    wget -q --show-progress "$SRC_URL" -O "${pkg}_source_archive"
    
    rm -rf "${pkg}_src"
    mkdir -p "${pkg}_src"
    tar -xf "${pkg}_source_archive" -C "${pkg}_src" 2>/dev/null || unzip -q "${pkg}_source_archive" -d "${pkg}_src" 2>/dev/null || tar -xJf "${pkg}_source_archive" -C "${pkg}_src" 2>/dev/null || true
    
    cd "${pkg}_src"
    cd "$(ls -d */ | head -n 1)"

    echo "==> Building $pkg using $NPROC core(s)..."
    if [ -n "$BUILD_CMD" ]; then
        eval "$BUILD_CMD"
    else
        ./configure --prefix="$PREFIX"
        make -j"$NPROC"
    fi
}

pm_install_only() {
    require_root
    local skip_confirm="false"
    if [ "$1" = "-s" ]; then
        skip_confirm="true"
        shift
    fi

    local pkg="$1"
    if [ -z "$pkg" ]; then
        echo "Error: No package specified to install." >&2
        exit 1
    fi

    local recipe_dir="$REPO_DIR/main/$pkg"
    local build_src="$BUILD_DIR/${pkg}_src"

    if [ ! -d "$build_src" ]; then
        echo "Error: No compiled build directory found for '$pkg'. Run 'pm b $pkg' first!" >&2
        exit 1
    fi

    confirm_action "Do you wish to install $pkg?" "$skip_confirm"

    local target_dir
    target_dir="$(find "$build_src" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
    if [ -z "$target_dir" ]; then
        target_dir="$build_src"
    fi
    cd "$target_dir"

    echo "==> Installing $pkg..."
    if [ -f "$recipe_dir/recipe" ]; then
        source "$recipe_dir/recipe"
    fi

    if [ -n "$INSTALL_CMD" ]; then
        eval "$INSTALL_CMD"
    else
        make install
    fi

    if ! grep -q "^${pkg}\$" "$INSTALLED_FILE" 2>/dev/null; then
        echo "$pkg" >> "$INSTALLED_FILE"
    fi

    echo "==> Successfully installed $pkg!"
}

pm_build_install() {
    require_root
    local skip_confirm="false"
    if [ "$1" = "-s" ]; then
        skip_confirm="true"
        shift
    fi

    local pkg="$1"
    if [ "$skip_confirm" = "true" ]; then
        pm_build -s "$pkg"
        pm_install_only -s "$pkg"
    else
        pm_build "$pkg"
        pm_install_only "$pkg"
    fi
}

pm_remove() {
    require_root
    local skip_confirm="false"
    if [ "$1" = "-s" ]; then
        skip_confirm="true"
        shift
    fi

    local pkg="$1"
    if [ -z "$pkg" ]; then
        echo "Error: No package specified to remove." >&2
        exit 1
    fi

    confirm_action "Do you wish to remove $pkg?" "$skip_confirm"

    local recipe_dir="$REPO_DIR/main/$pkg"
    local build_src="$BUILD_DIR/${pkg}_src"

    if [ -d "$build_src" ]; then
        cd "$build_src/$(ls -t "$build_src" | head -n 1)" 2>/dev/null || true
        if [ -f "$recipe_dir/recipe" ]; then
            source "$recipe_dir/recipe"
        fi

        echo "==> Removing $pkg..."
        if [ -n "$REMOVE_CMD" ]; then
            eval "$REMOVE_CMD"
        elif [ -f "Makefile" ]; then
            make uninstall 2>/dev/null || true
        fi
    fi

    echo "==> Cleaning up build junk for $pkg..."
    rm -rf "$BUILD_DIR/${pkg}_src" "$BUILD_DIR/${pkg}_source_archive"

    if [ -f "$INSTALLED_FILE" ]; then
        sed -i "/^${pkg}\$/d" "$INSTALLED_FILE"
    fi

    echo "==> Successfully removed $pkg and cleaned up."
}

case "$1" in
    s|u)
        pm_sync
        ;;
    re)
        pm_reinstall_self
        ;;
    c)
        pm_config
        ;;
    b)
        shift
        pm_build "$@"
        ;;
    bi)
        shift
        pm_build_install "$@"
        ;;
    i)
        shift
        pm_install_only "$@"
        ;;
    l)
        pm_list
        ;;
    r)
        shift
        pm_remove "$@"
        ;;
    h|--help|"")
        usage
        ;;
    *)
        usage
        ;;
esac
