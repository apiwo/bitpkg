package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	Prefix string
	PmHome string
	PmRepo string
	Nproc  string
}

func getRealHomeDir() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		if u, err := os.UserHomeDir(); err == nil && !strings.HasPrefix(u, "/root") {
			return u
		}
		return filepath.Join("/home", sudoUser)
	}
	if doasUser := os.Getenv("DOAS_USER"); doasUser != "" {
		return filepath.Join("/home", doasUser)
	}
	home, _ := os.UserHomeDir()
	return home
}

func getConfigPath() string {
	home := getRealHomeDir()
	return filepath.Join(home, ".config", "pm", "config")
}

func loadConfig() Config {
	home := getRealHomeDir()
	nprocDefault := strconv.Itoa(runtime.NumCPU())

	cfg := Config{
		Prefix: "/usr",
		PmHome: filepath.Join(home, ".local", "share", "pm"),
		PmRepo: "https://github.com/apiwo/pm.git",
		Nproc:  nprocDefault,
	}

	configFile := getConfigPath()
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			switch key {
			case "PREFIX":
				cfg.Prefix = val
			case "PM_HOME":
				cfg.PmHome = val
			case "PM_REPO":
				cfg.PmRepo = val
			case "NPROC":
				cfg.Nproc = val
			}
		}
	}

	return cfg
}

func saveConfig(cfg Config) {
	configFile := getConfigPath()
	os.MkdirAll(filepath.Dir(configFile), 0755)

	content := fmt.Sprintf("PREFIX=\"%s\"\nPM_HOME=\"%s\"\nPM_REPO=\"%s\"\nNPROC=\"%s\"\n",
		cfg.Prefix, cfg.PmHome, cfg.PmRepo, cfg.Nproc)

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n==> Configuration saved to %s\n", configFile)
}

func requireRoot() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Error: Root permissions needed.")
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("pm - A simple source-based package manager")
	fmt.Println("Usage:")
	fmt.Println("  pm, pm h             - Show this help menu")
	fmt.Println("  pm l                 - List installed packages")
	fmt.Println("  pm c                 - Open interactive configuration menu")
	fmt.Println("  pm s, pm u           - Sync / update local repository from GitHub (requires root)")
	fmt.Println("  pm re                - Reinstall / update 'pm' itself from GitHub (requires root)")
	fmt.Println("  pm b [-s] <pkgs...>  - Build package recipe(s) (requires root, -s to skip prompt)")
	fmt.Println("  pm bi [-s] <pkgs...> - Build and install package recipe(s) (requires root, -s to skip prompt)")
	fmt.Println("  pm i [-s] <pkgs...>  - Install already compiled package(s) (requires root, -s to skip prompt)")
	fmt.Println("  pm r [-s] <pkgs...>  - Remove package(s) and clean build artifacts (requires root, -s to skip prompt)")
	os.Exit(0)
}

func confirmAction(promptMsg string, skipFlag bool) bool {
	if skipFlag {
		return true
	}

	fmt.Printf("%s [Y/n]: ", promptMsg)
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if strings.ToLower(choice) == "n" || strings.ToLower(choice) == "no" {
		fmt.Println("==> Skipped by user.")
		return false
	}
	return true
}

func loadRecipe(recipePath string) map[string]string {
	vars := make(map[string]string)
	data, err := os.ReadFile(recipePath)
	if err != nil {
		return vars
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			vars[key] = val
		}
	}
	return vars
}

func executeShell(cmdStr string, dir string, envVars map[string]string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := os.Environ()
	for k, v := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	return cmd.Run()
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func parsePkgArgs(args []string) ([]string, bool) {
	skipConfirm := false
	var pkgs []string

	for _, arg := range args {
		if arg == "-s" {
			skipConfirm = true
		} else if strings.TrimSpace(arg) != "" {
			pkgs = append(pkgs, arg)
		}
	}
	return pkgs, skipConfirm
}

func findTargetDir(pkgSrcDir string) string {
	entries, err := os.ReadDir(pkgSrcDir)
	if err != nil {
		return pkgSrcDir
	}

	var subDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subDirs = append(subDirs, filepath.Join(pkgSrcDir, entry.Name()))
		}
	}

	if len(subDirs) == 1 {
		return findTargetDir(subDirs[0])
	}

	return pkgSrcDir
}

func pmConfig() {
	cfg := loadConfig()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=============================================")
	fmt.Println("         pm Configuration Menu               ")
	fmt.Println("=============================================")
	fmt.Println("")

	fmt.Printf("Install prefix directory [%s]: ", cfg.Prefix)
	inputPrefix, _ := reader.ReadString('\n')
	if val := strings.TrimSpace(inputPrefix); val != "" {
		cfg.Prefix = val
	}

	fmt.Printf("Package data directory [%s]: ", cfg.PmHome)
	inputHome, _ := reader.ReadString('\n')
	if val := strings.TrimSpace(inputHome); val != "" {
		cfg.PmHome = val
	}

	fmt.Printf("Package repository Git URL [%s]: ", cfg.PmRepo)
	inputRepo, _ := reader.ReadString('\n')
	if val := strings.TrimSpace(inputRepo); val != "" {
		cfg.PmRepo = val
	}

	maxCores := runtime.NumCPU()
	fmt.Printf("Number of cores to use for building (1-%d) [%s]: ", maxCores, cfg.Nproc)
	inputNproc, _ := reader.ReadString('\n')
	if val := strings.TrimSpace(inputNproc); val != "" {
		cfg.Nproc = val
	}

	saveConfig(cfg)
}

func pmList(cfg Config) {
	fmt.Println("==> Installed packages:")
	installedFile := filepath.Join(cfg.PmHome, "installed")
	data, err := os.ReadFile(installedFile)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		fmt.Println("  (No packages installed yet)")
		return
	}
	fmt.Print(string(data))
}

func pmSync(cfg Config) {
	requireRoot()
	repoDir := filepath.Join(cfg.PmHome, "repo")
	fmt.Printf("==> Syncing packages from %s...\n", cfg.PmRepo)

	// Automatically allow Git access under root/doas to prevent safe.directory ownership errors
	exec.Command("git", "config", "--global", "--add", "safe.directory", repoDir).Run()

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		cmd := exec.Command("git", "-C", repoDir, "pull")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	} else {
		cmd := exec.Command("git", "clone", cfg.PmRepo, repoDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
	fmt.Println("==> Sync complete.")
}

func pmReinstallSelf(cfg Config) {
	requireRoot()
	fmt.Println("==> Updating 'pm' itself from GitHub...")

	tempDir, err := os.MkdirTemp("", "pm_src_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("git", "clone", "--quiet", cfg.PmRepo, filepath.Join(tempDir, "pm_src"))
	cmd.Run()

	srcPath := filepath.Join(tempDir, "pm_src")
	if _, err := os.Stat(filepath.Join(srcPath, "main.go")); err == nil {
		buildCmd := exec.Command("go", "build", "-o", "/usr/bin/pm", filepath.Join(srcPath, "main.go"))
		buildCmd.Dir = srcPath
		buildCmd.Env = append(os.Environ(), "GO111MODULE=off")

		if err := buildCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error building updated pm binary: %v\n", err)
			os.Exit(1)
		}
		os.Chmod("/usr/bin/pm", 0755)
		fmt.Println("==> 'pm' has been successfully updated to the latest version!")
		fmt.Println("==> Your configuration file at ~/.config/pm/config remains intact.")
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, "Error: Could not find main.go in the repository root.")
		os.Exit(1)
	}
}

func buildPackage(cfg Config, pkg string, current int, total int, skipConfirm bool) bool {
	repoDir := filepath.Join(cfg.PmHome, "repo")
	buildDir := filepath.Join(cfg.PmHome, "build")
	recipeDir := filepath.Join(repoDir, "main", pkg)
	recipeFile := filepath.Join(recipeDir, "recipe")

	if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
		fmt.Printf("Error: Package '%s' not found in main/ directory!\n", pkg)
		return false
	}

	capName := capitalize(pkg)
	if !confirmAction(fmt.Sprintf("\nDo you wish to build %s?", pkg), skipConfirm) {
		return false
	}

	fmt.Printf("\n==> Building package (%d of %d) %s\n\n", current, total, capName)

	os.MkdirAll(buildDir, 0755)
	recipeVars := loadRecipe(recipeFile)
	srcURL := recipeVars["SRC_URL"]

	// Step 1: Fetching source
	fmt.Printf("=> Step (1 of 3) for package %s\n", capName)
	fmt.Println("> Fetching source")
	archivePath := filepath.Join(buildDir, fmt.Sprintf("%s_source_archive", pkg))
	exec.Command("wget", "-q", "--show-progress", srcURL, "-O", archivePath).Run()

	// Step 2: Unpacking
	fmt.Println("> Unpacking")
	pkgSrcDir := filepath.Join(buildDir, fmt.Sprintf("%s_src", pkg))
	os.RemoveAll(pkgSrcDir)
	os.MkdirAll(pkgSrcDir, 0755)

	tarCmd := exec.Command("tar", "-xf", archivePath, "-C", pkgSrcDir)
	if err := tarCmd.Run(); err != nil {
		unzipCmd := exec.Command("unzip", "-q", archivePath, "-d", pkgSrcDir)
		if err := unzipCmd.Run(); err != nil {
			exec.Command("tar", "-xJf", archivePath, "-C", pkgSrcDir).Run()
		}
	}

	targetDir := findTargetDir(pkgSrcDir)

	// Step 3: Building
	fmt.Println("> Building")
	envVars := map[string]string{
		"PREFIX": cfg.Prefix,
		"NPROC":  cfg.Nproc,
	}

	var buildErr error
	if buildCmd, ok := recipeVars["BUILD_CMD"]; ok && buildCmd != "" {
		buildErr = executeShell(buildCmd, targetDir, envVars)
	} else {
		buildErr = executeShell(fmt.Sprintf("./configure --prefix=%s && make -j%s", cfg.Prefix, cfg.Nproc), targetDir, envVars)
	}

	if buildErr != nil {
		fmt.Printf("Error building package %s: %v\n", pkg, buildErr)
		return false
	}

	return true
}

func installPackage(cfg Config, pkg string, skipConfirm bool) bool {
	repoDir := filepath.Join(cfg.PmHome, "repo")
	buildDir := filepath.Join(cfg.PmHome, "build")
	installedFile := filepath.Join(cfg.PmHome, "installed")
	recipeDir := filepath.Join(repoDir, "main", pkg)
	buildSrc := filepath.Join(buildDir, fmt.Sprintf("%s_src", pkg))

	if _, err := os.Stat(buildSrc); os.IsNotExist(err) {
		fmt.Printf("Error: No compiled build directory found for '%s'. Run build step first!\n", pkg)
		return false
	}

	capName := capitalize(pkg)
	if !confirmAction(fmt.Sprintf("\nDo you wish to install %s?", capName), skipConfirm) {
		return false
	}

	targetDir := findTargetDir(buildSrc)

	fmt.Printf("\n> Installing to \"%s\"\n", cfg.Prefix)
	recipeVars := loadRecipe(filepath.Join(recipeDir, "recipe"))
	envVars := map[string]string{
		"PREFIX": cfg.Prefix,
		"NPROC":  cfg.Nproc,
	}

	var installErr error
	if installCmd, ok := recipeVars["INSTALL_CMD"]; ok && installCmd != "" {
		installErr = executeShell(installCmd, targetDir, envVars)
	} else {
		installErr = executeShell("make install", targetDir, envVars)
	}

	if installErr != nil {
		fmt.Printf("Error installing package %s: %v\n", pkg, installErr)
		return false
	}

	installedData, _ := os.ReadFile(installedFile)
	lines := strings.Split(string(installedData), "\n")
	alreadyInstalled := false
	for _, line := range lines {
		if strings.TrimSpace(line) == pkg {
			alreadyInstalled = true
			break
		}
	}

	if !alreadyInstalled {
		f, err := os.OpenFile(installedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(pkg + "\n")
			f.Close()
		}
	}

	fmt.Printf("==> Successfully installed %s!\n", capName)
	return true
}

func pmBuild(cfg Config, args []string) {
	requireRoot()
	pkgs, skipConfirm := parsePkgArgs(args)
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No package specified.")
		os.Exit(1)
	}

	total := len(pkgs)
	for i, pkg := range pkgs {
		buildPackage(cfg, pkg, i+1, total, skipConfirm)
	}
}

func pmInstallOnly(cfg Config, args []string) {
	requireRoot()
	pkgs, skipConfirm := parsePkgArgs(args)
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No package specified to install.")
		os.Exit(1)
	}

	for _, pkg := range pkgs {
		installPackage(cfg, pkg, skipConfirm)
	}
}

func pmBuildInstall(cfg Config, args []string) {
	requireRoot()
	pkgs, skipConfirm := parsePkgArgs(args)
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No package specified.")
		os.Exit(1)
	}

	total := len(pkgs)
	for i, pkg := range pkgs {
		if buildPackage(cfg, pkg, i+1, total, skipConfirm) {
			installPackage(cfg, pkg, skipConfirm)
		}
	}
}

func pmRemove(cfg Config, args []string) {
	requireRoot()
	pkgs, skipConfirm := parsePkgArgs(args)
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No package specified to remove.")
		os.Exit(1)
	}

	repoDir := filepath.Join(cfg.PmHome, "repo")
	buildDir := filepath.Join(cfg.PmHome, "build")
	installedFile := filepath.Join(cfg.PmHome, "installed")

	total := len(pkgs)
	for i, pkg := range pkgs {
		capName := capitalize(pkg)
		if !confirmAction(fmt.Sprintf("\nDo you wish to remove %s?", capName), skipConfirm) {
			continue
		}

		fmt.Printf("\n==> Removing package (%d of %d) %s...\n", i+1, total, capName)

		recipeDir := filepath.Join(repoDir, "main", pkg)
		buildSrc := filepath.Join(buildDir, fmt.Sprintf("%s_src", pkg))

		if _, err := os.Stat(buildSrc); err == nil {
			targetDir := findTargetDir(buildSrc)

			recipeVars := loadRecipe(filepath.Join(recipeDir, "recipe"))
			envVars := map[string]string{
				"PREFIX": cfg.Prefix,
				"NPROC":  cfg.Nproc,
			}

			if removeCmd, ok := recipeVars["REMOVE_CMD"]; ok && removeCmd != "" {
				executeShell(removeCmd, targetDir, envVars)
			} else if _, err := os.Stat(filepath.Join(targetDir, "Makefile")); err == nil {
				executeShell("make uninstall", targetDir, envVars)
			}
		}

		fmt.Printf("==> Cleaning up build junk for %s...\n", capName)
		os.RemoveAll(buildSrc)
		os.RemoveAll(filepath.Join(buildDir, fmt.Sprintf("%s_source_archive", pkg)))

		if data, err := os.ReadFile(installedFile); err == nil {
			lines := strings.Split(string(data), "\n")
			var newLines []string
			for _, line := range lines {
				if strings.TrimSpace(line) != pkg && strings.TrimSpace(line) != "" {
					newLines = append(newLines, line)
				}
			}
			newContent := strings.Join(newLines, "\n")
			if len(newLines) > 0 {
				newContent += "\n"
			}
			os.WriteFile(installedFile, []byte(newContent), 0644)
		}

		fmt.Printf("==> Successfully removed %s!\n", capName)
	}
}

func main() {
	cfg := loadConfig()
	os.MkdirAll(cfg.PmHome, 0755)

	installedFile := filepath.Join(cfg.PmHome, "installed")
	if _, err := os.Stat(installedFile); os.IsNotExist(err) {
		os.WriteFile(installedFile, []byte(""), 0644)
	}

	if len(os.Args) < 2 {
		usage()
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "s", "u":
		pmSync(cfg)
	case "re":
		pmReinstallSelf(cfg)
	case "c":
		pmConfig()
	case "b":
		pmBuild(cfg, args)
	case "bi":
		pmBuildInstall(cfg, args)
	case "i":
		pmInstallOnly(cfg, args)
	case "l":
		pmList(cfg)
	case "r":
		pmRemove(cfg, args)
	case "h", "--help":
		usage()
	default:
		usage()
	}
}
