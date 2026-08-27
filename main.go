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

const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

type Config struct {
	Prefix  string
	BitHome string
	BitRepo string
	Nproc   string
	CC      string
	PkgMode string
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
	return filepath.Join(home, ".config", "bit", "config")
}

func loadConfig() Config {
	home := getRealHomeDir()
	nprocDefault := strconv.Itoa(runtime.NumCPU())

	cfg := Config{
		Prefix:  "/usr",
		BitHome: filepath.Join(home, ".local", "share", "bit"),
		BitRepo: "https://github.com/apiwo/bitpkg.git",
		Nproc:   nprocDefault,
		CC:      "gcc",
		PkgMode: "ask",
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
			case "BIT_HOME":
				cfg.BitHome = val
			case "BIT_REPO":
				cfg.BitRepo = val
			case "NPROC":
				cfg.Nproc = val
			case "CC":
				cfg.CC = val
			case "PKG_MODE":
				cfg.PkgMode = strings.ToLower(val)
			}
		}
	}
	return cfg
}

func saveConfig(cfg Config) {
	configFile := getConfigPath()
	os.MkdirAll(filepath.Dir(configFile), 0755)

	content := fmt.Sprintf("PREFIX=\"%s\"\nBIT_HOME=\"%s\"\nBIT_REPO=\"%s\"\nNPROC=\"%s\"\nCC=\"%s\"\nPKG_MODE=\"%s\"\n",
		cfg.Prefix, cfg.BitHome, cfg.BitRepo, cfg.Nproc, cfg.CC, cfg.PkgMode)

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError writing config: %v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}
	fmt.Printf("\n%s==>%s Configuration saved to %s\n", ColorGreen, ColorReset, configFile)
}

func requireRoot() {
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "%sError: Root permissions needed.%s\n", ColorRed, ColorReset)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("bit - Bit package manager (bitpkg)")
	fmt.Println("Usage:")
	fmt.Println("  bit, bit h               - Show this help menu")
	fmt.Println("  bit l                    - List installed packages")
	fmt.Println("  bit c                    - Open interactive configuration menu")
	fmt.Println("  bit s [-l], bit u [-l]   - Sync repository (-l to list synced packages)")
	fmt.Println("  bit q <query>            - Search repositories for a package (closest match)")
	fmt.Println("  bit re                   - Reinstall / update 'bit' itself from GitHub")
	fmt.Println("  bit b [-s] <pkgs...>     - Build package recipe(s)")
	fmt.Println("  bit bi [-s] <pkgs...>    - Build and install package recipe(s)")
	fmt.Println("  bit i [-s] <pkgs...>     - Install already compiled package(s)")
	fmt.Println("  bit r [-s] <pkgs...>     - Remove package(s) and clean build artifacts")
	fmt.Println("  bit fi [-b] <path>       - Install from local file/recipe (-b for binary archive)")
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
		fmt.Printf("%s==>%s Skipped by user.\n", ColorYellow, ColorReset)
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

func isBinaryInstalled(dep string) bool {
	_, err := exec.LookPath(dep)
	return err == nil
}

func isBitInstalled(cfg Config, dep string) bool {
	installedFile := filepath.Join(cfg.BitHome, "installed")
	data, err := os.ReadFile(installedFile)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == dep {
			return true
		}
	}
	return false
}

func markInstalled(cfg Config, pkg string) {
	installedFile := filepath.Join(cfg.BitHome, "installed")
	data, _ := os.ReadFile(installedFile)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == pkg {
			return
		}
	}
	f, err := os.OpenFile(installedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(pkg + "\n")
		f.Close()
	}
}

func determineInstallMode(cfg Config, pkg string, skipConfirm bool) string {
	binaryPath := filepath.Join(cfg.BitHome, "repo", "binary", pkg, fmt.Sprintf("%s.tar.xz", pkg))
	hasBinary := true
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		hasBinary = false
	}

	mode := cfg.PkgMode
	if mode == "ask" && hasBinary {
		if skipConfirm {
			return "binary"
		}
		fmt.Printf("Default package? Binary, Source, Ask.\nFound binary for %s. Install binary? [Y/n]: ", pkg)
		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(choice)) == "n" {
			return "source"
		}
		return "binary"
	}

	if mode == "binary" && !hasBinary {
		fmt.Printf("%s==>%s Binary requested but not found for %s, falling back to source.\n", ColorYellow, ColorReset, pkg)
		return "source"
	}

	return mode
}

func installBinaryDirect(cfg Config, pkg string) bool {
	binaryPath := filepath.Join(cfg.BitHome, "repo", "binary", pkg, fmt.Sprintf("%s.tar.xz", pkg))
	fmt.Printf("\n%s==>%s Installing binary package %s to %s\n", ColorGreen, ColorReset, pkg, cfg.Prefix)

	cmd := exec.Command("tar", "-xJf", binaryPath, "-C", cfg.Prefix)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("%sError extracting binary %s: %v%s\n", ColorRed, pkg, err, ColorReset)
		return false
	}

	markInstalled(cfg, pkg)
	fmt.Printf("%s==>%s Successfully installed %s!\n", ColorGreen, ColorReset, pkg)
	return true
}

func resolveDependencies(cfg Config, pkg string, skipConfirm bool, visited map[string]bool) bool {
	if visited[pkg] {
		return true
	}
	visited[pkg] = true

	recipeFile := filepath.Join(cfg.BitHome, "repo", "main", pkg, "recipe")
	if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
		recipeFile = filepath.Join(cfg.BitHome, "repo", "repo", "main", pkg, "recipe")
	}

	vars := loadRecipe(recipeFile)
	if strings.TrimSpace(vars["DEPS"]) == "" {
		return true
	}

	var missing []string
	for _, dep := range strings.Fields(vars["DEPS"]) {
		if !isBinaryInstalled(dep) && !isBitInstalled(cfg, dep) {
			missing = append(missing, dep)
		}
	}
	if len(missing) == 0 {
		fmt.Printf("%s==>%s Dependencies can be satisfied\n", ColorGreen, ColorReset)
		return true
	}

	var installable, unresolvable []string
	for _, dep := range missing {
		depRecipe := filepath.Join(cfg.BitHome, "repo", "main", dep, "recipe")
		if _, err := os.Stat(depRecipe); os.IsNotExist(err) {
			depRecipe = filepath.Join(cfg.BitHome, "repo", "repo", "main", dep, "recipe")
		}
		
		if _, err := os.Stat(depRecipe); os.IsNotExist(err) {
			fmt.Printf("%s==>%s Interrupt: Dependency %s not found on system cannot be satisfied.\n", ColorRed, ColorReset, dep)
			unresolvable = append(unresolvable, dep)
		} else {
			installable = append(installable, dep)
		}
	}
	
	if len(unresolvable) > 0 {
		fmt.Printf("%s==>%s Missing dependencies: %s\n", ColorRed, ColorReset, strings.Join(unresolvable, ", "))
		fmt.Printf("%s==>%s Fatal: Task cannot be completed.\n", ColorRed, ColorReset)
		return false
	}

	fmt.Printf("%s==>%s Satisfying dependencies for %s...\n", ColorGreen, ColorReset, pkg)
	for i, dep := range installable {
		fmt.Printf("%s==>%s Installing dependency (%d of %d) for package %s\n", ColorGreen, ColorReset, i+1, len(installable), pkg)
		if !resolveDependencies(cfg, dep, skipConfirm, visited) {
			return false
		}

		mode := determineInstallMode(cfg, dep, skipConfirm)
		if mode == "binary" {
			installBinaryDirect(cfg, dep)
		} else {
			if buildPackage(cfg, dep, 1, 1, skipConfirm, false) {
				installPackage(cfg, dep, skipConfirm)
			}
		}
	}
	return true
}

func bitConfig() {
	cfg := loadConfig()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=============================================")
	fmt.Println("         bit Configuration Menu              ")
	fmt.Println("=============================================")

	fmt.Printf("Install prefix directory [%s]: ", cfg.Prefix)
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		cfg.Prefix = strings.TrimSpace(val)
	}

	fmt.Printf("Package data directory [%s]: ", cfg.BitHome)
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		cfg.BitHome = strings.TrimSpace(val)
	}

	fmt.Printf("Compiler (gcc, clang, tcc) [%s]: ", cfg.CC)
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		cfg.CC = strings.TrimSpace(val)
	}

	fmt.Printf("Default mode (binary, source, ask) [%s]: ", cfg.PkgMode)
	if val, _ := reader.ReadString('\n'); strings.TrimSpace(val) != "" {
		cfg.PkgMode = strings.ToLower(strings.TrimSpace(val))
	}

	saveConfig(cfg)
}

func bitList(cfg Config) {
	data, err := os.ReadFile(filepath.Join(cfg.BitHome, "installed"))
	if err != nil {
		fmt.Printf("\n%s==>%s Installed Packages:\n", ColorCyan, ColorBold)
		fmt.Println("--------------------------------------------------")
		fmt.Println("  (No packages installed yet)")
		fmt.Println("--------------------------------------------------\n")
		return
	}

	lines := strings.Split(string(data), "\n")
	var packages []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			packages = append(packages, trimmed)
		}
	}

	if len(packages) == 0 {
		fmt.Printf("\n%s==>%s Installed Packages:\n", ColorCyan, ColorBold)
		fmt.Println("--------------------------------------------------")
		fmt.Println("  (No packages installed yet)")
		fmt.Println("--------------------------------------------------\n")
		return
	}

	fmt.Printf("\n%s==>%s Installed Packages (%d total):%s\n", ColorCyan, ColorBold, len(packages), ColorReset)
	fmt.Println("--------------------------------------------------")
	for _, pkg := range packages {
		fmt.Printf("  %s•%s %s%s%s\n", ColorGreen, ColorReset, ColorBold, pkg, ColorReset)
	}
	fmt.Println("--------------------------------------------------\n")
}

func bitSync(cfg Config, showList bool) {
	requireRoot()
	repoDir := filepath.Join(cfg.BitHome, "repo")
	fmt.Printf("%s==>%s Syncing packages from %s...\n", ColorGreen, ColorReset, cfg.BitRepo)
	exec.Command("git", "config", "--global", "--add", "safe.directory", repoDir).Run()

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		exec.Command("git", "-C", repoDir, "pull").Run()
	} else {
		exec.Command("git", "clone", cfg.BitRepo, repoDir).Run()
	}
	fmt.Printf("%s==>%s Sync complete.\n", ColorGreen, ColorReset)

	if showList {
		listSyncedPackages(cfg)
	}
}

// Levenshtein distance helper for closest matching
func levenshteinDistance(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range t {
		d[0][j] = j
	}
	for i := 1; i <= len(s); i++ {
		for j := 1; j <= len(t); j++ {
			cost := 1
			if s[i-1] == t[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,
				min(d[i][j-1]+1, d[i-1][j-1]+cost),
			)
		}
	}
	return d[len(s)][len(t)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func bitQuery(cfg Config, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: bit q <query>")
		return
	}
	query := strings.ToLower(strings.TrimSpace(args[0]))
	repoDir := filepath.Join(cfg.BitHome, "repo")

	mainDir := filepath.Join(repoDir, "main")
	if _, err := os.Stat(mainDir); os.IsNotExist(err) {
		mainDir = filepath.Join(repoDir, "repo", "main")
	}
	binDir := filepath.Join(repoDir, "binary")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		binDir = filepath.Join(repoDir, "repo", "binary")
	}

	packages := make(map[string]map[string]bool) // pkgName -> type (source/binary) -> true

	if entries, err := os.ReadDir(mainDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				if packages[name] == nil {
					packages[name] = make(map[string]bool)
				}
				packages[name]["source"] = true
			}
		}
	}

	if entries, err := os.ReadDir(binDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				if packages[name] == nil {
					packages[name] = make(map[string]bool)
				}
				packages[name]["binary"] = true
			}
		}
	}

	if len(packages) == 0 {
		fmt.Printf("%s==>%s No packages found in repository. Try running 'bit s' first.\n", ColorYellow, ColorReset)
		return
	}

	// Exact match check first
	if types, exists := packages[query]; exists {
		var typeList []string
		if types["source"] {
			typeList = append(typeList, "Source")
		}
		if types["binary"] {
			typeList = append(typeList, "Binary")
		}
		fmt.Printf("%s==>%s Package found: %s%s%s [Type: %s]\n", ColorGreen, ColorReset, ColorBold, query, ColorReset, strings.Join(typeList, ", "))
		return
	}

	// Find closest match using Levenshtein distance
	closestPkg := ""
	minDist := 999999
	for pkg := range packages {
		dist := levenshteinDistance(query, pkg)
		if dist < minDist {
			minDist = dist
			closestPkg = pkg
		}
	}

	if closestPkg != "" {
		types := packages[closestPkg]
		var typeList []string
		if types["source"] {
			typeList = append(typeList, "Source")
		}
		if types["binary"] {
			typeList = append(typeList, "Binary")
		}
		fmt.Printf("%s==>%s Package '%s' not found. Did you mean:\n", ColorYellow, ColorReset, query)
		fmt.Printf("     %s%s%s [Type: %s] (Distance: %d)\n", ColorBold, closestPkg, ColorReset, strings.Join(typeList, ", "), minDist)
	} else {
		fmt.Printf("%s==>%s No matching packages found for '%s'.\n", ColorRed, ColorReset, query)
	}
}

func listSyncedPackages(cfg Config) {
	repoDir := filepath.Join(cfg.BitHome, "repo")

	repoListPath := filepath.Join(repoDir, "repo.list")
	if data, err := os.ReadFile(repoListPath); err == nil {
		fmt.Printf("\n%s==>%s Repository Package Manifest (repo.list):%s\n", ColorCyan, ColorBold, ColorReset)
		fmt.Println("--------------------------------------------------")
		lines := strings.Split(string(data), "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				fmt.Printf("  • %s\n", strings.TrimSpace(l))
			}
		}
		fmt.Println("--------------------------------------------------")
		return
	}

	fmt.Printf("\n%s==>%s Available Repository Packages:%s\n", ColorCyan, ColorBold, ColorReset)
	mainDir := filepath.Join(repoDir, "main")
	if _, err := os.Stat(mainDir); os.IsNotExist(err) {
		mainDir = filepath.Join(repoDir, "repo", "main")
	}
	binDir := filepath.Join(repoDir, "binary")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		binDir = filepath.Join(repoDir, "repo", "binary")
	}

	entries, err := os.ReadDir(mainDir)
	if err == nil {
		fmt.Printf("\n%s[ Source Recipes ]%s\n", ColorBold, ColorReset)
		for _, e := range entries {
			if e.IsDir() {
				fmt.Printf("  • %s\n", e.Name())
			}
		}
	}

	binEntries, err := os.ReadDir(binDir)
	if err == nil {
		fmt.Printf("\n%s[ Binary Archives ]%s\n", ColorBold, ColorReset)
		for _, e := range binEntries {
			if e.IsDir() {
				fmt.Printf("  • %s\n", e.Name())
			}
		}
	}
	fmt.Println()
}

func bitReinstallSelf(cfg Config) {
	requireRoot()
	fmt.Printf("%s==>%s Updating 'bit' itself from GitHub...\n", ColorGreen, ColorReset)
	tempDir, _ := os.MkdirTemp("", "bit_src_*")
	defer os.RemoveAll(tempDir)

	exec.Command("git", "clone", "--quiet", cfg.BitRepo, filepath.Join(tempDir, "src")).Run()
	srcPath := filepath.Join(tempDir, "src")

	buildCmd := exec.Command("go", "build", "-o", "/usr/bin/bit", "main.go")
	buildCmd.Dir = srcPath

	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError building bit: %v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}
	os.Chmod("/usr/bin/bit", 0755)

	fmt.Printf("%s==>%s 'bit' successfully updated!\n", ColorGreen, ColorReset)
	os.Exit(0)
}

func buildPackage(cfg Config, pkg string, current int, total int, skipConfirm bool, isFileInstall bool) bool {
	recipeFile := filepath.Join(cfg.BitHome, "repo", "main", pkg, "recipe")
	if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
		recipeFile = filepath.Join(cfg.BitHome, "repo", "repo", "main", pkg, "recipe")
	}
	if isFileInstall {
		recipeFile = pkg
	}

	if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
		fmt.Printf("%sError: Recipe not found!%s\n", ColorRed, ColorReset)
		return false
	}

	if !confirmAction(fmt.Sprintf("\nDo you wish to build %s?", pkg), skipConfirm) {
		return false
	}
	fmt.Printf("\n%s==>%s Building package (%d of %d) %s%s%s\n\n", ColorGreen, ColorReset, current, total, ColorBold, pkg, ColorReset)

	buildDir := filepath.Join(cfg.BitHome, "build")
	os.MkdirAll(buildDir, 0755)

	recipeVars := loadRecipe(recipeFile)
	if !isFileInstall && !resolveDependencies(cfg, pkg, skipConfirm, make(map[string]bool)) {
		return false
	}

	archivePath := filepath.Join(buildDir, fmt.Sprintf("%s_source_archive", filepath.Base(pkg)))
	exec.Command("wget", "-q", "--show-progress", recipeVars["SRC_URL"], "-O", archivePath).Run()

	pkgSrcDir := filepath.Join(buildDir, fmt.Sprintf("%s_src", filepath.Base(pkg)))
	os.RemoveAll(pkgSrcDir)
	os.MkdirAll(pkgSrcDir, 0755)

	if err := exec.Command("tar", "-xf", archivePath, "-C", pkgSrcDir).Run(); err != nil {
		if err := exec.Command("unzip", "-q", archivePath, "-d", pkgSrcDir).Run(); err != nil {
			exec.Command("tar", "-xJf", archivePath, "-C", pkgSrcDir).Run()
		}
	}

	targetDir := pkgSrcDir
	entries, _ := os.ReadDir(pkgSrcDir)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(pkgSrcDir, e.Name()))
		}
	}
	if len(dirs) == 1 {
		targetDir = dirs[0]
	}

	envVars := map[string]string{"PREFIX": cfg.Prefix, "NPROC": cfg.Nproc, "CC": cfg.CC}

	var buildErr error
	if cmd, ok := recipeVars["BUILD_CMD"]; ok {
		buildErr = executeShell(cmd, targetDir, envVars)
	} else {
		buildErr = executeShell("CC=$CC ./configure --prefix=$PREFIX && make -j$NPROC", targetDir, envVars)
	}

	if buildErr != nil {
		fmt.Printf("%sError building %s: %v%s\n", ColorRed, pkg, buildErr, ColorReset)
		return false
	}
	return true
}

func installPackage(cfg Config, pkg string, skipConfirm bool) bool {
	if !confirmAction(fmt.Sprintf("\nDo you wish to install %s?", pkg), skipConfirm) {
		return false
	}

	buildSrc := filepath.Join(cfg.BitHome, "build", fmt.Sprintf("%s_src", filepath.Base(pkg)))
	targetDir := buildSrc
	if entries, _ := os.ReadDir(buildSrc); len(entries) > 0 {
		var dirs []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(buildSrc, e.Name()))
			}
		}
		if len(dirs) == 1 {
			targetDir = dirs[0]
		}
	}

	recipeFile := filepath.Join(cfg.BitHome, "repo", "main", pkg, "recipe")
	if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
		recipeFile = filepath.Join(cfg.BitHome, "repo", "repo", "main", pkg, "recipe")
	}
	if _, err := os.Stat(recipeFile); err != nil {
		recipeFile = pkg
	}

	envVars := map[string]string{"PREFIX": cfg.Prefix, "NPROC": cfg.Nproc, "CC": cfg.CC}
	recipeVars := loadRecipe(recipeFile)

	var err error
	if cmd, ok := recipeVars["INSTALL_CMD"]; ok {
		err = executeShell(cmd, targetDir, envVars)
	} else {
		err = executeShell("make install", targetDir, envVars)
	}

	if err != nil {
		fmt.Printf("%sError installing: %v%s\n", ColorRed, err, ColorReset)
		return false
	}

	markInstalled(cfg, filepath.Base(pkg))
	fmt.Printf("%s==>%s Successfully installed %s!\n", ColorGreen, ColorReset, pkg)
	return true
}

func bitRemove(cfg Config, pkgs []string, skipConfirm bool) {
	requireRoot()
	if len(pkgs) == 0 {
		os.Exit(1)
	}
	for _, pkg := range pkgs {
		if !confirmAction(fmt.Sprintf("Remove %s?", pkg), skipConfirm) {
			continue
		}

		buildSrc := filepath.Join(cfg.BitHome, "build", fmt.Sprintf("%s_src", pkg))
		targetDir := buildSrc
		if entries, _ := os.ReadDir(buildSrc); len(entries) > 0 {
			var dirs []string
			for _, e := range entries {
				if e.IsDir() {
					dirs = append(dirs, filepath.Join(buildSrc, e.Name()))
				}
			}
			if len(dirs) == 1 {
				targetDir = dirs[0]
			}
		}

		recipeFile := filepath.Join(cfg.BitHome, "repo", "main", pkg, "recipe")
		if _, err := os.Stat(recipeFile); os.IsNotExist(err) {
			recipeFile = filepath.Join(cfg.BitHome, "repo", "repo", "main", pkg, "recipe")
		}

		vars := loadRecipe(recipeFile)
		envVars := map[string]string{"PREFIX": cfg.Prefix}
		if cmd, ok := vars["REMOVE_CMD"]; ok {
			executeShell(cmd, targetDir, envVars)
		} else {
			executeShell("make uninstall", targetDir, envVars)
		}

		os.RemoveAll(buildSrc)

		data, _ := os.ReadFile(filepath.Join(cfg.BitHome, "installed"))
		var newLines []string
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) != pkg && strings.TrimSpace(l) != "" {
				newLines = append(newLines, l)
			}
		}
		os.WriteFile(filepath.Join(cfg.BitHome, "installed"), []byte(strings.Join(newLines, "\n")+"\n"), 0644)
		fmt.Printf("%s==>%s Removed %s\n", ColorGreen, ColorReset, pkg)
	}
}

func bitFileInstall(cfg Config, args []string) {
	requireRoot()
	if len(args) == 0 {
		return
	}

	isBinary := false
	var target string
	if args[0] == "-b" {
		if len(args) < 2 {
			return
		}
		isBinary = true
		target = args[1]
	} else {
		target = args[0]
	}

	if isBinary {
		fmt.Printf("\n%s==>%s Extracting binary %s to %s\n", ColorGreen, ColorReset, target, cfg.Prefix)
		exec.Command("tar", "-xJf", target, "-C", cfg.Prefix).Run()
		markInstalled(cfg, filepath.Base(target))
	} else {
		if buildPackage(cfg, target, 1, 1, true, true) {
			installPackage(cfg, target, true)
		}
	}
}

func main() {
	cfg := loadConfig()
	os.MkdirAll(cfg.BitHome, 0755)

	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "s", "u":
		showList := false
		for _, arg := range args {
			if arg == "-l" {
				showList = true
			}
		}
		bitSync(cfg, showList)
	case "q":
		bitQuery(cfg, args)
	case "re":
		bitReinstallSelf(cfg)
	case "c":
		bitConfig()
	case "l":
		bitList(cfg)
	case "fi":
		bitFileInstall(cfg, args)
	case "b", "bi", "i", "r":
		requireRoot()
		pkgs, skip := parsePkgArgs(args)
		for _, pkg := range pkgs {
			if cmd == "r" {
				bitRemove(cfg, []string{pkg}, skip)
				continue
			}

			mode := determineInstallMode(cfg, pkg, skip)
			if mode == "binary" {
				installBinaryDirect(cfg, pkg)
			} else {
				if cmd == "b" || cmd == "bi" {
					if buildPackage(cfg, pkg, 1, 1, skip, false) && cmd == "bi" {
						installPackage(cfg, pkg, skip)
					}
				} else if cmd == "i" {
					installPackage(cfg, pkg, skip)
				}
			}
		}
	default:
		usage()
	}
}
