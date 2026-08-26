# Bit Package Manager (bitpkg)

Bit is a fast, lightweight, source-and-binary package manager written in Go. It was designed to manage custom package repositories, automatically resolve dependencies, and give users total control over compilation.

## Features
* **Hybrid Management**: Build from source using recipes, or drop-in pre-compiled `.tar.xz` binary archives.
* **Custom Compilers**: Easily swap between `gcc`, `clang`, `tcc`, or cross-compilers via the config menu.
* **Smart Dependencies**: Recursively resolves and builds missing dependencies before compilation.
* **Root Safety**: Drops to standard user context for Git syncs but manages strict root access for `PREFIX` modifications.

## Installation

Run the provided bootstrap script as root:

    sudo ./install.sh

The script will compile the `bit` binary, set up the user configurations (`~/.config/bit`), and initialize the package data store.

## Repository Structure

To host your own packages, structure your git repository like this:

    repo/
    ├── main/
    │   ├── htop/
    │   │   └── recipe
    │   └── nano/
    │       └── recipe
    └── binary/
        ├── htop/
        │   └── htop.tar.xz
        └── nano/
            └── nano.tar.xz

## Binary Archive Format

A binary package (e.g., `htop.tar.xz`) must be structured exactly how it should appear on the root filesystem under the `$PREFIX`. If `$PREFIX=/usr`, your tarball contents should look like this:

    bin/htop
    share/man/man1/htop.1

## Usage

* `bit s` - Sync the remote repository.
* `bit bi <package>` - Build and install a package.
* `bit c` - Configure settings (Compiler, Prefix, etc).
* `bit fi -b /path/to/archive.tar.xz` - Install a local binary archive directly.
