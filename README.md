# pm

`pm` is a simple, source-based package manager written in Go. It builds software packages directly from source recipes and manages installation, updates, and removal.

## Requirements

- Go (1.20 or later)
- Git
- Wget
- Tar / Unzip
- Make (and common build toolchains like gcc, clang, etc.)

## Installation

Clone the repository and run the installation script with root permissions:

    git clone https://github.com/apiwo/pm.git
    cd pm
    chmod +x install.sh
    ./install.sh

The installer compiles `main.go`, prompts for initial configuration setup, installs the executable to `/usr/bin/pm`, and syncs the package repository.

## Usage

    pm [command] [options] [package]

### Commands

| Command | Description |
| :--- | :--- |
| `pm`, `pm h` | Display help menu |
| `pm l` | List installed packages |
| `pm c` | Open interactive configuration menu |
| `pm s`, `pm u` | Sync / update local recipe repository from GitHub (requires root) |
| `pm re` | Reinstall / update `pm` itself from GitHub (requires root) |
| `pm b [-s] <pkg>` | Build package recipe without installing (requires root) |
| `pm bi [-s] <pkg>` | Build and install package recipe (requires root) |
| `pm i [-s] <pkg>` | Install an already compiled package (requires root) |
| `pm r [-s] <pkg>` | Remove package and clean build artifacts (requires root) |

### Options

- `-s`: Skip confirmation prompts.

## Package Recipes

Recipes are stored in the repository under `main/<pkg>/recipe`. A recipe defines environment variables used during build and installation:

    SRC_URL="https://example.com/downloads/pkg-1.0.tar.gz"
    BUILD_CMD="./configure --prefix=$PREFIX && make -j$NPROC"
    INSTALL_CMD="make install"
    REMOVE_CMD="make uninstall"

If `BUILD_CMD`, `INSTALL_CMD`, or `REMOVE_CMD` are omitted, `pm` defaults to standard `./configure`, `make -j$NPROC`, `make install`, and `make uninstall`.

## Configuration

Configuration is saved at `~/.config/pm/config`. You can reconfigure settings at any time by running:

    pm c

Default values:

- `PREFIX`: `/usr`
- `PM_HOME`: `~/.local/share/pm`
- `PM_REPO`: `https://github.com/apiwo/pm.git`
- `NPROC`: Number of CPU cores available on the system
