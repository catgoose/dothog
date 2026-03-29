//go:build mage

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"catgoose/dothog/internal/setup"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default Environment Variables
var (
	env        = envOr("ENV", "development")
	envFile    = fmt.Sprintf(".env.%s", env)
	binaryName = "dothog"
	proxyHost  = "localhost"
	buildPath  = "build"
	binPath    = "./bin"
	// The following ports are templated by setup (internal/setup or mage setup):
	// - APP_TLS_PORT: Echo TLS port (SERVER_LISTEN_PORT)
	// - TEMPL_HTTP_PORT: templ's local HTTP proxy port
	// - CADDY_TLS_PORT: Caddy TLS termination port
	proxyURL               = fmt.Sprintf("https://%s:{{APP_TLS_PORT}}", proxyHost)
	proxyPort              = "{{TEMPL_HTTP_PORT}}"
	caddyTLSPort           = "{{CADDY_TLS_PORT}}"
	htmxURL                = "https://unpkg.com/htmx.org"
	htmxResponseTargetsURL = "https://unpkg.com/htmx-ext-response-targets"
	htmxSSEURL             = "https://unpkg.com/htmx-ext-sse"
	hyperscriptURL         = "https://unpkg.com/hyperscript.org"
	alpineURL              = "https://unpkg.com/alpinejs@3/dist/cdn.min.js"
	alpineMorphURL         = "https://unpkg.com/@alpinejs/morph@3/dist/cdn.min.js"
	htmxAlpineMorphURL     = "https://unpkg.com/htmx-ext-alpine-morph@2.0.0/alpine-morph.js"
	publicSourceDir        = "web/assets/public"
	publicOutputDir        = filepath.Join(buildPath, publicSourceDir)
	publicJSDir            = filepath.Join(publicSourceDir, "js")
	publicCSSDir           = filepath.Join(publicSourceDir, "css")
)

// DaisyUIComponents is the list of DaisyUI CSS component URLs.
var daisyUIComponents = []string{
	"npm/daisyui@5/base/rootscrollgutter.css",
	"npm/daisyui@5/base/reset.css",
	"npm/daisyui@5/base/rootcolor.css",
	"npm/daisyui@5/base/scrollbar.css",
	"npm/daisyui@5/base/svg.css",
	"npm/daisyui@5/base/rootscrolllock.css",
	"npm/daisyui@5/base/properties.css",
	"npm/daisyui@5/components/checkbox.css",
	"npm/daisyui@5/components/menu.css",
	"npm/daisyui@5/components/input.css",
	"npm/daisyui@5/components/select.css",
	"npm/daisyui@5/components/button.css",
	"npm/daisyui@5/components/toggle.css",
	"npm/daisyui@5/theme/light.css",
	"npm/daisyui@5/theme/dark.css",
	"npm/daisyui@5/theme/cupcake.css",
	"npm/daisyui@5/theme/emerald.css",
	"npm/daisyui@5/theme/corporate.css",
	"npm/daisyui@5/theme/synthwave.css",
	"npm/daisyui@5/theme/retro.css",
	"npm/daisyui@5/theme/cyberpunk.css",
	"npm/daisyui@5/theme/valentine.css",
	"npm/daisyui@5/theme/garden.css",
	"npm/daisyui@5/theme/forest.css",
	"npm/daisyui@5/theme/lofi.css",
	"npm/daisyui@5/theme/pastel.css",
	"npm/daisyui@5/theme/fantasy.css",
	"npm/daisyui@5/theme/wireframe.css",
	"npm/daisyui@5/theme/luxury.css",
	"npm/daisyui@5/theme/dracula.css",
	"npm/daisyui@5/theme/cmyk.css",
	"npm/daisyui@5/theme/autumn.css",
	"npm/daisyui@5/theme/business.css",
	"npm/daisyui@5/theme/acid.css",
	"npm/daisyui@5/theme/lemonade.css",
	"npm/daisyui@5/theme/night.css",
	"npm/daisyui@5/theme/coffee.css",
	"npm/daisyui@5/theme/winter.css",
	"npm/daisyui@5/theme/dim.css",
	"npm/daisyui@5/theme/nord.css",
	"npm/daisyui@5/theme/sunset.css",
	"npm/daisyui@5/theme/caramellatte.css",
	"npm/daisyui@5/theme/abyss.css",
	"npm/daisyui@5/theme/silk.css",
}

// Helper function to get environment variable with default
func envOr(env, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

func init() {
	_ = os.Setenv("MAGEFILE_VERBOSE", "1")
}

// serverListenPortFromEnvFile reads SERVER_LISTEN_PORT from a key=value env file.
func serverListenPortFromEnvFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SERVER_LISTEN_PORT=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "SERVER_LISTEN_PORT="))
		}
	}
	return ""
}

// resolvePort returns p unchanged if it looks like a real port (no setup placeholder).
// When the template hasn't been set up yet, it derives the port from
// SERVER_LISTEN_PORT in the env file, adding offset (0=app, 1=templ, 2=caddy).
func resolvePort(p string, offset int) string {
	if !strings.Contains(p, "{{") {
		return p
	}
	if base := serverListenPortFromEnvFile(envFile); base != "" {
		if n, err := strconv.Atoi(base); err == nil {
			return strconv.Itoa(n + offset)
		}
	}
	return p
}

// resolveProxyURL returns the proxy URL, deriving the port from the env file
// when the setup placeholder hasn't been replaced yet.
func resolveProxyURL() string {
	if !strings.Contains(proxyURL, "{{") {
		return proxyURL
	}
	if base := serverListenPortFromEnvFile(envFile); base != "" {
		if _, err := strconv.Atoi(base); err == nil {
			return fmt.Sprintf("https://%s:%s", proxyHost, base)
		}
	}
	return proxyURL
}

// Tailwind runs the Tailwind CSS compilation
func Tailwind() error {
	return sh.Run(filepath.Join(binPath, "tailwindcss"),
		"-i", "web/styles/input.css",
		"-o", filepath.Join(publicCSSDir, "tailwind.css"),
		"-m")
}

// TailwindWatch runs Tailwind in watch mode
func TailwindWatch() error {
	if _, err := os.Stat(filepath.Join(binPath, "tailwindcss")); os.IsNotExist(err) {
		fmt.Println("Tailwind binary not found. Running update...")
		mg.Deps(TailwindUpdate)
	}
	return sh.Run(filepath.Join(binPath, "tailwindcss"),
		"-i", "web/styles/input.css",
		"-o", filepath.Join(publicCSSDir, "tailwind.css"),
		"-m", "-w")
}

// TailwindUpdate downloads the Tailwind CLI from GitHub releases.
func TailwindUpdate() error {
	mg.Deps(PrepareDirs)
	assetName := tailwindAssetName()
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/tailwindlabs/tailwindcss/releases/latest", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github API returned %s", resp.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}
	if release.TagName == "" {
		return fmt.Errorf("no tag_name in release")
	}
	downloadURL := fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/%s/%s", release.TagName, assetName)
	resp2, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp2.Status)
	}
	binDir := filepath.Join(binPath, ".")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	outPath := filepath.Join(binPath, assetName)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, resp2.Body)
	out.Close()
	if err != nil {
		os.Remove(outPath)
		return err
	}
	if err := os.Chmod(outPath, 0755); err != nil {
		return err
	}
	linkPath := filepath.Join(binPath, "tailwindcss")
	if runtime.GOOS != "windows" {
		os.Remove(linkPath)
		if err := os.Symlink(assetName, linkPath); err != nil {
			return err
		}
	} else {
		data, _ := os.ReadFile(outPath)
		if err := os.WriteFile(linkPath, data, 0755); err != nil {
			return err
		}
	}
	return nil
}

func tailwindAssetName() string {
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			return "tailwindcss-linux-x64"
		}
		return "tailwindcss-linux-" + runtime.GOARCH
	case "darwin":
		return "tailwindcss-macos-" + runtime.GOARCH
	case "windows":
		return "tailwindcss-windows-x64.exe"
	default:
		return "tailwindcss-linux-x64"
	}
}

// DaisyUpdate updates DaisyUI CSS
func DaisyUpdate() error {
	mg.Deps(PrepareDirs)
	daisyURL := fmt.Sprintf("https://cdn.jsdelivr.net/combine/%s",
		joinURLs(daisyUIComponents))
	return downloadFile(daisyURL, filepath.Join(publicCSSDir, "daisyui.css"))
}

// HtmxUpdate updates HTMX and all HTMX extension files
func HtmxUpdate() error {
	mg.Deps(PrepareDirs)
	if err := downloadFile(htmxURL, filepath.Join(publicJSDir, "htmx.min.js")); err != nil {
		return err
	}
	if err := downloadFile(htmxResponseTargetsURL, filepath.Join(publicJSDir, "htmx.response-targets.js")); err != nil {
		return err
	}
	return downloadFile(htmxSSEURL, filepath.Join(publicJSDir, "htmx.ext.sse.js"))
}

// HyperscriptUpdate updates Hyperscript file
func HyperscriptUpdate() error {
	return downloadFile(hyperscriptURL, filepath.Join(publicJSDir, "_hyperscript.min.js"))
}

// AlpineUpdate downloads Alpine.js core, the morph plugin, and the htmx alpine-morph extension.
func AlpineUpdate() error {
	mg.Deps(PrepareDirs)
	if err := downloadFile(alpineURL, filepath.Join(publicJSDir, "alpine.min.js")); err != nil {
		return err
	}
	if err := downloadFile(alpineMorphURL, filepath.Join(publicJSDir, "alpine.morph.min.js")); err != nil {
		return err
	}
	return downloadFile(htmxAlpineMorphURL, filepath.Join(publicJSDir, "htmx.alpine-morph.js"))
}

// UpdateAssets updates all assets
func UpdateAssets() error {
	if err := HyperscriptUpdate(); err != nil {
		return fmt.Errorf("hyperscript update failed: %v", err)
	}
	if err := HtmxUpdate(); err != nil {
		return fmt.Errorf("htmx update failed: %v", err)
	}
	if err := AlpineUpdate(); err != nil {
		return fmt.Errorf("alpine update failed: %v", err)
	}
	if err := DaisyUpdate(); err != nil {
		return fmt.Errorf("daisy update failed: %v", err)
	}
	if err := TailwindUpdate(); err != nil {
		return fmt.Errorf("tailwind update failed: %v", err)
	}
	return nil
}

// Air runs the Air live reload tool
func Air() error {
	fmt.Println("running air")

	return sh.Run("go", "tool", "air",
		"-c", ".air/server.toml",
		"-build.cmd", fmt.Sprintf("go build -o ./tmp/main . && %s",
			getTemplNotifyProxyCmd()))
}

// Templ runs Templ in watch mode
func Templ() error {
	return TemplWatch()
}

// TemplWatch runs Templ in watch mode
func TemplWatch() error {
	cmd := getTemplCmd()
	return sh.RunV(cmd[0], cmd[1:]...)
}

// TemplGenerate generates Templ files
func TemplGenerate() error {
	return sh.Run("go", "tool", "templ", "generate")
}

// Clean removes build and debug files
func Clean() error {
	if err := CleanBuild(); err != nil {
		return fmt.Errorf("clean build failed: %v", err)
	}
	if err := CleanDebug(); err != nil {
		return fmt.Errorf("clean debug failed: %v", err)
	}
	return nil
}

// CleanBuild removes the build directory
func CleanBuild() error {
	return os.RemoveAll(buildPath)
}

// CleanDebug removes debug binaries
func CleanDebug() error {
	matches, err := filepath.Glob("__debug_bin*")
	if err != nil {
		return err
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return err
		}
	}
	return nil
}

// PrepareDirs creates necessary directories
func PrepareDirs() error {
	dirs := []string{
		publicOutputDir,
		publicJSDir,
		publicCSSDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// CopyFiles copies necessary files to build directory
func CopyFiles() error {
	if err := EnvCheck(); err != nil {
		return fmt.Errorf("environment check failed: %v", err)
	}

	if err := PrepareDirs(); err != nil {
		return fmt.Errorf("prepare directories failed: %v", err)
	}

	// Copy env file
	if err := sh.Copy(filepath.Join(buildPath, filepath.Base(envFile)), envFile); err != nil {
		return fmt.Errorf("failed to copy env file: %v", err)
	}

	if err := copyDir("web/views", filepath.Join(buildPath, "web/views")); err != nil {
		return fmt.Errorf("failed to copy views directory: %v", err)
	}

	dirs := []string{
		buildPath,
		filepath.Join(buildPath, "web"),
		filepath.Join(buildPath, "web/assets"),
		publicOutputDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	if err := copyDir(publicSourceDir, publicOutputDir); err != nil {
		return fmt.Errorf("failed to copy public assets: %v", err)
	}

	return nil
}

// Compile builds the Go project
func Compile() error {
	ldflags := "-w -X catgoose/dothog/internal/version.BuildDate=" + time.Now().UTC().Format("2006-01-02")
	return sh.Run("go", "build",
		"-ldflags", ldflags,
		"-o", filepath.Join(buildPath, binaryName),
		"main.go")
}

// Run executes the compiled binary
func Run() error {
	mg.Deps(Build)
	return sh.Run(filepath.Join(buildPath, binaryName))
}

// EnvCheck verifies the environment file exists
func EnvCheck() error {
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("error: %s file not found", envFile)
	}
	return nil
}

// Watch runs Tailwind, Templ, Air, and Caddy in watch mode
func Watch() error {
	// Signal to the Go server that it's behind the templ proxy chain.
	// Echo skips gzip middleware when this is set — Caddy handles compression.
	os.Setenv("TEMPL_PROXY", "true")
	errc := make(chan error, 4)

	go func() {
		errc <- TailwindWatch()
	}()

	go func() {
		errc <- TemplWatch()
	}()

	go func() {
		errc <- Air()
	}()

	go func() {
		errc <- CaddyStart()
	}()

	if os.Getenv("OPEN_BROWSER") != "false" {
		go func() {
			time.Sleep(2 * time.Second)
			url := "https://localhost:" + resolvePort(caddyTLSPort, 2)
			fmt.Println("Opening browser at:", url)
			openBrowserURL(url)
		}()
	}

	// Wait for all commands to complete or error
	for range 4 {
		if err := <-errc; err != nil {
			return err
		}
	}
	return nil
}

// openBrowserURL opens the given URL in the default browser.
func openBrowserURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func nodeModulesCheck() error {
	if _, err := os.Stat("node_modules"); os.IsNotExist(err) {
		return errors.New("node_modules not found. Run: npm ci")
	}
	return nil
}

// Build cleans, updates assets, and builds the project
func Build() error {
	fmt.Println("Starting build process...")

	if err := nodeModulesCheck(); err != nil {
		return err
	}

	if err := Clean(); err != nil {
		return fmt.Errorf("clean failed: %v", err)
	}
	fmt.Println("✓ Clean completed")

	if err := Tailwind(); err != nil {
		return fmt.Errorf("tailwind compilation failed: %v", err)
	}
	fmt.Println("✓ Tailwind compiled")

	if err := Compile(); err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}
	fmt.Println("✓ Project compiled")

	if err := CopyFiles(); err != nil {
		return fmt.Errorf("copy files failed: %v", err)
	}
	fmt.Println("✓ Files copied")

	fmt.Println("Build completed successfully")
	return nil
}

// Helper function to join URLs for DaisyUI
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.Create(destPath)
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		out.Close()
		return err
	})
}

func joinURLs(urls []string) string {
	result := ""
	for i, url := range urls {
		if i > 0 {
			result += ","
		}
		result += url
	}
	return result
}

// Helper function to get Templ command
func getTemplCmd() []string {
	return []string{
		"go", "tool", "templ", "generate",
		"-watch",
		"-open-browser=false",
		"-proxy=" + resolveProxyURL(),
		"-proxybind=" + proxyHost,
		"-proxyport=" + resolvePort(proxyPort, 1),
	}
}

// Helper function to get Templ notify proxy command
func getTemplNotifyProxyCmd() string {
	return fmt.Sprintf("go tool templ generate --notify-proxy -proxy=%s -proxybind=%s -proxyport=%s",
		resolveProxyURL(), proxyHost, resolvePort(proxyPort, 1))
}

// Helper function to download a file
func downloadFile(url, filepath string) error {
	return sh.Run("curl", "-Lso", filepath, url)
}

// setup:feature:demo:start

// SetupTo copies the template to dest, runs setup there, and leaves this repo
// untouched. Use it to preview exactly what a consumer gets after running setup.
//
// Usage:
//
//	mage setupTo /tmp/myapp "My App Name"
//	SETUP_MODULE=github.com/me/myapp SETUP_PORT=12345 mage setupTo /tmp/myapp "My App"
//
// Env vars (all optional):
//
//	SETUP_MODULE  Go module path for the new app (default: auto-derived from app name)
//	SETUP_PORT    5-digit base port number (default: random)
func SetupTo(dest, appName string) error {
	ctx := context.Background()

	src, err := os.Getwd()
	if err != nil {
		return err
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absDest); err == nil {
		fmt.Printf("Removing existing directory: %s\n", absDest)
		if err := os.RemoveAll(absDest); err != nil {
			return fmt.Errorf("remove %s: %w", absDest, err)
		}
	}

	// Ensure node_modules are present before copying
	if _, err := os.Stat(filepath.Join(src, "package-lock.json")); err == nil {
		if err := sh.Run("npm", "ci"); err != nil {
			return fmt.Errorf("npm ci: %w", err)
		}
	}

	fmt.Printf("Copying template to %s...\n", absDest)
	if err := setup.CopyRepoTo(src, absDest, []string{".git", "bin", "build", "tmp"}); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	opts := setup.Options{
		AppName:    appName,
		ModulePath: envOr("SETUP_MODULE", ""),
		BasePort:   envOr("SETUP_PORT", ""),
	}
	if featuresEnv := os.Getenv("SETUP_FEATURES"); featuresEnv != "" {
		opts.Features = parseFeatureFlag(featuresEnv)
	}
	fmt.Printf("Running setup (app=%q, module=%q, port=%q, features=%v)...\n",
		opts.AppName, opts.ModulePath, opts.BasePort, opts.Features)
	if err := setup.Run(ctx, absDest, opts); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	fmt.Printf("\n✓ Setup complete → %s\n", absDest)
	fmt.Printf("  cd %s && go build ./...\n", absDest)
	return nil
}

// setup:feature:demo:end

// parseFeatureFlag parses the --features value.
// "all" → all features, "none" → empty slice, otherwise comma-separated tags.
func parseFeatureFlag(val string) []string {
	val = strings.TrimSpace(val)
	switch strings.ToLower(val) {
	case "all":
		return append([]string{}, setup.AllFeatures...)
	case "none":
		return []string{}
	}
	var features []string
	for _, f := range strings.Split(val, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			features = append(features, f)
		}
	}
	// Auto-include dependencies
	hasSSE, hasCaddy, hasAvatar, hasGraph := false, false, false, false
	for _, f := range features {
		switch f {
		case setup.FeatureSSE:
			hasSSE = true
		case setup.FeatureCaddy:
			hasCaddy = true
		case setup.FeatureAvatar:
			hasAvatar = true
		case setup.FeatureGraph:
			hasGraph = true
		}
	}
	if hasSSE && !hasCaddy {
		features = append(features, setup.FeatureCaddy)
		fmt.Println("SSE requires Caddy for proxying; Caddy auto-included.")
	}
	if hasAvatar && !hasGraph {
		features = append(features, setup.FeatureGraph)
		fmt.Println("Avatar requires Graph API; Graph auto-included.")
	}
	return features
}

// Lint runs static analysis and style checks on the codebase.
func Lint() error {
	// Check if golangci-lint is available
	if _, err := sh.Exec(nil, nil, nil, "which", "golangci-lint"); err != nil {
		return errors.New("golangci-lint not found. Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	}
	fmt.Println("Running golangci-lint...")
	if err := sh.RunV("golangci-lint", "run"); err != nil {
		return err
	}

	// Check if golint is available
	if _, err := sh.Exec(nil, nil, nil, "which", "golint"); err != nil {
		return errors.New("golint not found. Please install it: go install golang.org/x/lint/golint@latest")
	}
	fmt.Println("Running golint...")
	if err := sh.RunV("golint", "./..."); err != nil {
		return err
	}

	// Check if fieldalignment is available
	if _, err := sh.Exec(nil, nil, nil, "which", "fieldalignment"); err != nil {
		return errors.New("fieldalignment not found. Please install it: go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest")
	}
	fmt.Println("Running fieldalignment...")
	if err := sh.RunV("fieldalignment", "./..."); err != nil {
		return err
	}

	return nil
}

// FixFieldAlignment runs fieldalignment with the -fix flag to automatically fix field alignment issues
func FixFieldAlignment() error {
	// Check if fieldalignment is available
	if _, err := sh.Exec(nil, nil, nil, "which", "fieldalignment"); err != nil {
		return errors.New("fieldalignment not found. Please install it: go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest")
	}
	fmt.Println("Running fieldalignment with -fix...")
	return sh.RunV("fieldalignment", "-fix", "./...")
}

// LintWatch runs Air with lint configuration for automatic linting on file changes
func LintWatch() error {
	fmt.Println("Starting Air lint watch mode...")
	fmt.Println("Press Ctrl+C to stop")
	return sh.Run("air", "-c", ".air/lint.toml")
}

// Test runs all tests
func Test() error {
	fmt.Println("Running tests...")
	return sh.RunV("go", "test", "./...")
}

// TestVerbose runs all tests with verbose output
func TestVerbose() error {
	fmt.Println("Running tests with verbose output...")
	return sh.RunV("go", "test", "-v", "./...")
}

// TestCoverage runs tests with coverage report
func TestCoverage() error {
	fmt.Println("Running tests with coverage...")
	return sh.RunV("go", "test", "-cover", "./...")
}

// TestCoverageHTML runs tests and generates HTML coverage report
func TestCoverageHTML() error {
	fmt.Println("Running tests with HTML coverage report...")
	if err := sh.RunV("go", "test", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
}

// TestBenchmark runs benchmark tests
func TestBenchmark() error {
	fmt.Println("Running benchmark tests...")
	return sh.RunV("go", "test", "-bench=.", "./...")
}

// TestRace runs tests with race detection
func TestRace() error {
	fmt.Println("Running tests with race detection...")
	return sh.RunV("go", "test", "-race", "./...")
}

// TestE2E runs Playwright end-to-end tests
func TestE2E() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Running Playwright e2e tests...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts")
}

// TestE2EHeaded runs Playwright tests in headed browser mode
func TestE2EHeaded() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Running Playwright e2e tests (headed)...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts", "--headed")
}

// TestE2EUI opens the Playwright interactive UI
func TestE2EUI() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Opening Playwright UI...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts", "--ui")
}

// TestWatch runs tests in watch mode using the Go-based watcher
func TestWatch() error {
	fmt.Println("Building and starting Go test watcher...")
	fmt.Println("Tests will run automatically on .go file changes")
	fmt.Println("Press Ctrl+C to stop")

	// Build the test watcher
	if err := sh.Run("go", "build", "-o", filepath.Join(binPath, "testwatcher"), "./cmd/testwatcher"); err != nil {
		return fmt.Errorf("failed to build test watcher: %w", err)
	}

	// Run the test watcher
	return sh.Run(filepath.Join(binPath, "testwatcher"))
}

// CaddyInstall installs Caddy for local development
func CaddyInstall() error {
	fmt.Println("Installing Caddy...")
	return sh.Run("go", "install", "github.com/caddyserver/caddy/v2/cmd/caddy@latest")
}

// CaddyStart starts Caddy with the local Caddyfile.
// When the template hasn't been set up yet the Caddyfile still contains
// {{CADDY_TLS_PORT}} / {{TEMPL_HTTP_PORT}} placeholders that Caddy can't
// parse.  We resolve them to real port numbers and write the result to
// tmp/Caddyfile so Caddy always receives a valid config.
func CaddyStart() error {
	caddyfile := filepath.Join("config", "Caddyfile")
	if _, err := os.Stat(caddyfile); os.IsNotExist(err) {
		fmt.Println("Caddyfile not found, skipping Caddy.")
		return nil
	}
	resolvedCaddyPort := resolvePort(caddyTLSPort, 2)
	resolvedTemplPort := resolvePort(proxyPort, 1)

	fmt.Println("Starting Caddy with TLS termination...")
	fmt.Println("Access your app at: https://localhost:" + resolvedCaddyPort)
	fmt.Println("Press Ctrl+C to stop")

	data, err := os.ReadFile(caddyfile)
	if err != nil {
		return fmt.Errorf("read Caddyfile: %w", err)
	}
	content := strings.ReplaceAll(string(data), "{{CADDY_TLS_PORT}}", resolvedCaddyPort)
	content = strings.ReplaceAll(content, "{{TEMPL_HTTP_PORT}}", resolvedTemplPort)

	if err := os.MkdirAll("tmp", 0755); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	tmpCaddyfile := filepath.Join("tmp", "Caddyfile")
	if err := os.WriteFile(tmpCaddyfile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write tmp Caddyfile: %w", err)
	}

	return sh.Run("caddy", "run", "--config", tmpCaddyfile)
}

// setup:feature:capacitor:start

// IosDeps installs Capacitor iOS dependencies (requires macOS with Xcode).
func IosDeps() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("iOS targets require macOS with Xcode installed")
	}
	// Add the iOS platform if not already present
	if _, err := os.Stat("ios"); os.IsNotExist(err) {
		if err := sh.Run("npx", "cap", "add", "ios"); err != nil {
			return fmt.Errorf("cap add ios: %w", err)
		}
	}
	return nil
}

// IosSync copies web assets and Capacitor config to the iOS project.
func IosSync() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("iOS targets require macOS with Xcode installed")
	}
	return sh.Run("npx", "cap", "sync", "ios")
}

// IosOpen opens the iOS project in Xcode.
func IosOpen() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("iOS targets require macOS with Xcode installed")
	}
	return sh.Run("npx", "cap", "open", "ios")
}

// IosRun builds and runs the app in the iOS simulator.
func IosRun() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("iOS targets require macOS with Xcode installed")
	}
	mg.Deps(IosSync)
	return sh.Run("npx", "cap", "run", "ios")
}

// IosBeta builds and uploads to TestFlight via Fastlane (requires macOS).
func IosBeta() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("iOS targets require macOS with Xcode installed")
	}
	mg.Deps(IosSync)
	return sh.Run("bundle", "exec", "fastlane", "beta")
}

// setup:feature:capacitor:end
