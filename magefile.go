//go:build mage

package main

import (
	"bufio"
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

var (
	env        = envOr("ENV", "development")
	envFile    = fmt.Sprintf(".env.%s", env)
	binaryName = "dothog"
	proxyHost  = "localhost"
	buildPath  = "build"
	binPath    = "./bin"
	// The following ports are templated by setup (internal/setup or mage setup):
	// - APP_HTTP_PORT: Echo origin port (SERVER_LISTEN_PORT) — plain HTTP in dev
	// - TEMPL_HTTP_PORT: templ's local HTTP proxy port
	// - CADDY_TLS_PORT: Caddy TLS termination port (only when caddy feature is selected)
	proxyURL  = fmt.Sprintf("http://%s:{{APP_HTTP_PORT}}", proxyHost)
	proxyPort = "{{TEMPL_HTTP_PORT}}"
	// setup:feature:caddy:start
	caddyTLSPort = "{{CADDY_TLS_PORT}}"
	// setup:feature:caddy:end
	htmxURL                = "https://unpkg.com/htmx.org"
	htmxResponseTargetsURL = "https://unpkg.com/htmx-ext-response-targets"
	// setup:feature:sse:start
	htmxSSEURL = "https://unpkg.com/htmx-ext-sse"
	// setup:feature:sse:end
	// setup:feature:demo:start
	tavernJSURL = "https://cdn.jsdelivr.net/gh/catgoose/tavern-js@latest/dist/tavern.min.js"
	// setup:feature:demo:end
	hyperscriptURL     = "https://unpkg.com/hyperscript.org"
	alpineURL          = "https://unpkg.com/@alpinejs/csp@3/dist/cdn.min.js"
	alpineMorphURL     = "https://unpkg.com/@alpinejs/morph@3/dist/cdn.min.js"
	htmxAlpineMorphURL = "https://unpkg.com/htmx-ext-alpine-morph@2.0.0/alpine-morph.js"
	publicSourceDir    = "web/assets/public"
	publicOutputDir    = filepath.Join(buildPath, publicSourceDir)
	publicJSDir        = filepath.Join(publicSourceDir, "js")
	publicCSSDir       = filepath.Join(publicSourceDir, "css")
)

const templWatchIgnorePattern = `(^|[\\/])mage_output_file\.go$`
const caddyModuleVersion = "v2.11.3"

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
			return fmt.Sprintf("http://%s:%s", proxyHost, base)
		}
	}
	return proxyURL
}

// tailwindLocalBin returns the path to the locally-installed Tailwind CLI,
// with `.exe` appended on Windows so os/exec can resolve it.
func tailwindLocalBin() string {
	bin := filepath.Join(binPath, "tailwindcss")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	return bin
}

// hostBinaryExt returns the executable extension for the host platform.
func hostBinaryExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func caddyLocalBin() string {
	return filepath.Join(binPath, "caddy"+hostBinaryExt())
}

func installRepoLocalCaddy(projectDir string) error {
	absBinDir, err := filepath.Abs(filepath.Join(projectDir, binPath))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absBinDir, 0755); err != nil {
		return err
	}
	return sh.RunWithV(
		map[string]string{"GOBIN": absBinDir},
		"go", "install", "github.com/caddyserver/caddy/v2/cmd/caddy@"+caddyModuleVersion,
	)
}

func confirmInstallCaddy() (bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false, fmt.Errorf("caddy binary not found and no interactive terminal is available; run `go tool mage caddyinstall` to install ./bin/caddy")
	}
	fmt.Print("Caddy not found in ./bin or PATH. Install Caddy into ./bin now? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func resolveCaddyBinary() (string, error) {
	if _, err := os.Stat(caddyLocalBin()); err == nil {
		return caddyLocalBin(), nil
	}
	if path, err := exec.LookPath("caddy"); err == nil {
		return path, nil
	}
	install, err := confirmInstallCaddy()
	if err != nil {
		return "", err
	}
	if !install {
		return "", fmt.Errorf("caddy binary not found; run `go tool mage caddyinstall` to install ./bin/caddy")
	}
	fmt.Println("Installing Caddy into ./bin...")
	if err := installRepoLocalCaddy("."); err != nil {
		return "", err
	}
	return caddyLocalBin(), nil
}

// Tailwind compiles web/styles/input.css to the public tailwind.css bundle (minified).
func Tailwind() error {
	return sh.Run(tailwindLocalBin(),
		"-i", "web/styles/input.css",
		"-o", filepath.Join(publicCSSDir, "tailwind.css"),
		"-m")
}

// TailwindWatch runs the Tailwind compiler in watch mode, installing the CLI first if missing.
func TailwindWatch() error {
	if _, err := os.Stat(tailwindLocalBin()); os.IsNotExist(err) {
		fmt.Println("Tailwind binary not found. Running update...")
		mg.Deps(TailwindUpdate)
	}
	return sh.Run(tailwindLocalBin(),
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
	linkPath := tailwindLocalBin()
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

// DaisyUpdate downloads the combined DaisyUI CSS bundle into public/css.
func DaisyUpdate() error {
	mg.Deps(PrepareDirs)
	daisyURL := fmt.Sprintf("https://cdn.jsdelivr.net/combine/%s",
		joinURLs(daisyUIComponents))
	return downloadFile(daisyURL, filepath.Join(publicCSSDir, "daisyui.css"))
}

// HtmxUpdate downloads HTMX core and the response-targets/SSE extensions into public/js.
func HtmxUpdate() error {
	mg.Deps(PrepareDirs)
	if err := downloadFile(htmxURL, filepath.Join(publicJSDir, "htmx.min.js")); err != nil {
		return err
	}
	if err := downloadFile(htmxResponseTargetsURL, filepath.Join(publicJSDir, "htmx.response-targets.js")); err != nil {
		return err
	}
	// setup:feature:sse:start
	if err := downloadFile(htmxSSEURL, filepath.Join(publicJSDir, "htmx.ext.sse.js")); err != nil {
		return err
	}
	// setup:feature:sse:end
	return nil
}

// HyperscriptUpdate downloads the _hyperscript runtime into public/js.
func HyperscriptUpdate() error {
	return downloadFile(hyperscriptURL, filepath.Join(publicJSDir, "_hyperscript.min.js"))
}

// setup:feature:demo:start

// TavernJSUpdate downloads the tavern-js SSE companion library.
func TavernJSUpdate() error {
	return downloadFile(tavernJSURL, filepath.Join(publicJSDir, "tavern.min.js"))
}

// setup:feature:demo:end

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

// UpdateAssets downloads every vendored client asset (HTMX, Alpine, Hyperscript, DaisyUI, Tailwind, tavern-js).
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
	// setup:feature:demo:start
	if err := TavernJSUpdate(); err != nil {
		return fmt.Errorf("tavern-js update failed: %v", err)
	}
	// setup:feature:demo:end
	return nil
}

// Air watches Go sources via .air/server.toml and rebuilds tmp/main on change.
func Air() error {
	fmt.Println("running air")
	return sh.Run("go", "tool", "air", "-c", ".air/server.toml")
}

// AirBuild performs Air's rebuild step without relying on shell operators.
// CGO_ENABLED is set via the process env so the same target works on Windows.
func AirBuild() error {
	if err := sh.RunWithV(map[string]string{"CGO_ENABLED": "0"},
		"go", "build", "-o", "./tmp/main"+hostBinaryExt(), "."); err != nil {
		return err
	}
	cmd := getTemplNotifyProxyArgs()
	return sh.RunV(cmd[0], cmd[1:]...)
}

// Templ is an alias for TemplWatch.
func Templ() error {
	return TemplWatch()
}

// TemplWatch runs `templ generate -watch` behind a proxy on TEMPL_HTTP_PORT.
func TemplWatch() error {
	cmd := getTemplCmd()
	return sh.RunV(cmd[0], cmd[1:]...)
}

// TemplGenerate produces *_templ.go in one shot; use TemplWatch for live regen.
func TemplGenerate() error {
	return sh.Run("go", "tool", "templ", "generate")
}

// Clean removes the build/ output tree and any Delve __debug_bin* binaries.
func Clean() error {
	if err := CleanBuild(); err != nil {
		return fmt.Errorf("clean build failed: %v", err)
	}
	if err := CleanDebug(); err != nil {
		return fmt.Errorf("clean debug failed: %v", err)
	}
	return nil
}

// CleanBuild deletes build/ recursively (the deployable-output tree).
func CleanBuild() error {
	return os.RemoveAll(buildPath)
}

// CleanDebug removes Delve __debug_bin* binaries left in the working tree.
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

// PrepareDirs ensures build/, build/public/js, and build/public/css exist
// before Tailwind/Templ/CopyFiles write into them.
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

// CopyFiles copies the env file, web/views, and public assets into build/.
func CopyFiles() error {
	if err := EnvCheck(); err != nil {
		return fmt.Errorf("environment check failed: %v", err)
	}

	if err := PrepareDirs(); err != nil {
		return fmt.Errorf("prepare directories failed: %v", err)
	}

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

// Compile builds main.go into build/<binaryName> with BuildDate baked in via ldflags.
func Compile() error {
	ldflags := "-w -X catgoose/dothog/internal/version.BuildDate=" + time.Now().UTC().Format("2006-01-02")
	return sh.Run("go", "build",
		"-ldflags", ldflags,
		"-o", filepath.Join(buildPath, binaryName+hostBinaryExt()),
		"main.go")
}

// Run executes the release-like artifact under build/, rebuilding it first via Build.
func Run() error {
	mg.Deps(Build)
	return sh.Run(filepath.Join(buildPath, binaryName+hostBinaryExt()))
}

// EnvCheck fails fast when the selected .env.<ENV> file is missing.
func EnvCheck() error {
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("error: %s file not found", envFile)
	}
	return nil
}

// Watch runs Tailwind, Templ, Air, and (when enabled) Caddy concurrently for local dev; opens the browser unless OPEN_BROWSER=false.
func Watch() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}

	type task struct {
		name string
		fn   func() error
	}
	tasks := []task{
		{"tailwind", TailwindWatch},
		{"templ", TemplWatch},
		{"air", Air},
		// setup:feature:caddy:start
		{"caddy", CaddyStart},
		// setup:feature:caddy:end
	}

	// Signal to the Go server that it's behind the templ HTTP proxy in
	// `mage watch`. Echo skips gzip middleware and 103 Early Hints when this
	// is set — those don't survive the templ proxy (or Caddy in front of it).
	os.Setenv("TEMPL_PROXY", "true")

	errc := make(chan error, len(tasks))
	for _, t := range tasks {
		go func() {
			errc <- t.fn()
		}()
	}

	if os.Getenv("OPEN_BROWSER") != "false" {
		go func() {
			time.Sleep(2 * time.Second)
			url := "http://localhost:" + resolvePort(proxyPort, 1)
			// setup:feature:caddy:start
			url = "https://localhost:" + resolvePort(caddyTLSPort, 2)
			// setup:feature:caddy:end
			fmt.Println("Opening browser at:", url)
			openBrowserURL(url)
		}()
	}

	for range len(tasks) {
		if err := <-errc; err != nil {
			return err
		}
	}
	return nil
}

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
		fmt.Println("node_modules not found, running npm ci...")
		if err := sh.Run("npm", "ci"); err != nil {
			return fmt.Errorf("npm ci failed: %w", err)
		}
	}
	return nil
}

// Build runs npm ci (if needed), Clean, Tailwind, Compile, then CopyFiles to produce a deployable build/ tree.
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

func getTemplCmd() []string {
	return []string{
		"go", "tool", "templ", "generate",
		"-watch",
		"-ignore-pattern=" + templWatchIgnorePattern,
		"-open-browser=false",
		"-proxy=" + resolveProxyURL(),
		"-proxybind=" + proxyHost,
		"-proxyport=" + resolvePort(proxyPort, 1),
	}
}

func getTemplNotifyProxyArgs() []string {
	return []string{
		"go", "tool", "templ", "generate",
		"--notify-proxy",
		"-proxy=" + resolveProxyURL(),
		"-proxybind=" + proxyHost,
		"-proxyport=" + resolvePort(proxyPort, 1),
	}
}

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

	fmt.Printf("Copying template to %s...\n", absDest)
	if err := setup.CopyRepoTo(src, absDest, []string{".git", ".claude", ".cursor", "bin", "build", "log", "node_modules", "test-results", "tmp", "localhost.crt", "localhost.key"}); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	opts := setup.Options{
		AppName:    appName,
		ModulePath: envOr("SETUP_MODULE", ""),
		BasePort:   envOr("SETUP_PORT", ""),
		Platform:   envOr("SETUP_PLATFORM", ""),
	}
	if featuresEnv := os.Getenv("SETUP_FEATURES"); featuresEnv != "" {
		opts.Features = parseFeatureFlag(featuresEnv)
	}
	fmt.Printf("Running setup (app=%q, module=%q, port=%q, platform=%q, features=%v)...\n",
		opts.AppName, opts.ModulePath, opts.BasePort, opts.Platform, opts.Features)
	if err := setup.Run(ctx, absDest, opts); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	fmt.Printf("\n✓ Setup complete → %s\n", absDest)
	fmt.Printf("  cd %s && go build ./...\n", absDest)
	return nil
}

// setup:feature:demo:end

// setup:feature:demo:start

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

// setup:feature:demo:end

// runQuiet runs a command with stdout/stderr piped through without the verbose exec: prefix.
func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Lint runs golangci-lint, golint, fieldalignment, and (when installed) oxlint.
func Lint() error {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return errors.New("golangci-lint not found. Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	}
	fmt.Println("Running golangci-lint...")
	if err := runQuiet("golangci-lint", "run"); err != nil {
		return err
	}

	if _, err := exec.LookPath("golint"); err != nil {
		return errors.New("golint not found. Please install it: go install golang.org/x/lint/golint@latest")
	}
	fmt.Println("Running golint...")
	if err := runQuiet("golint", "./..."); err != nil {
		return err
	}

	if _, err := exec.LookPath("fieldalignment"); err != nil {
		return errors.New("fieldalignment not found. Please install it: go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest")
	}
	fmt.Println("Running fieldalignment...")
	if err := runFilteredFieldAlignment(); err != nil {
		return err
	}

	// oxlint is optional — warn if not installed
	if _, err := exec.LookPath("oxlint"); err != nil {
		fmt.Println("Warning: oxlint not found, skipping JS lint. Install: curl -sSf https://oxc.rs/install.sh | sh")
	} else {
		fmt.Println("Running oxlint...")
		if err := runQuiet("oxlint", "web/assets/public/js/"); err != nil {
			return err
		}
	}

	return nil
}

// runFilteredFieldAlignment runs fieldalignment and filters out _templ.go generated files.
func runFilteredFieldAlignment() error {
	out, err := exec.Command("fieldalignment", "./...").CombinedOutput()
	if err == nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.Contains(line, "_templ.go") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return nil
	}
	fmt.Println(strings.Join(lines, "\n"))
	return errors.New("fieldalignment issues found")
}

// FixFieldAlignment runs `fieldalignment -fix` to rewrite struct field order in place.
func FixFieldAlignment() error {
	if _, err := exec.LookPath("fieldalignment"); err != nil {
		return errors.New("fieldalignment not found. Please install it: go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest")
	}
	fmt.Println("Running fieldalignment with -fix...")
	return runQuiet("fieldalignment", "-fix", "./...")
}

// LintWatch runs Air with .air/lint.toml to re-lint on file changes.
func LintWatch() error {
	fmt.Println("Starting Air lint watch mode...")
	fmt.Println("Press Ctrl+C to stop")
	return sh.Run("go", "tool", "air", "-c", ".air/lint.toml")
}

// Test is the broadest Go test target; it executes the full package tree.
func Test() error {
	fmt.Println("Running tests...")
	return sh.RunV("go", "test", "./...")
}

// TestVerbose adds per-test logging to the full package-tree test run.
func TestVerbose() error {
	fmt.Println("Running tests with verbose output...")
	return sh.RunV("go", "test", "-v", "./...")
}

// TestCoverage reports statement coverage across the full package tree.
func TestCoverage() error {
	fmt.Println("Running tests with coverage...")
	return sh.RunV("go", "test", "-cover", "./...")
}

// TestCoverageHTML produces coverage.out and renders it to coverage.html.
func TestCoverageHTML() error {
	fmt.Println("Running tests with HTML coverage report...")
	if err := sh.RunV("go", "test", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	return sh.RunV("go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
}

// TestBenchmark enables Go benchmarks across the package tree.
func TestBenchmark() error {
	fmt.Println("Running benchmark tests...")
	return sh.RunV("go", "test", "-bench=.", "./...")
}

// TestRace enables the race detector across the full package tree.
func TestRace() error {
	fmt.Println("Running tests with race detection...")
	return sh.RunV("go", "test", "-race", "./...")
}

// TestE2E runs Playwright e2e tests headlessly using e2e/playwright.config.ts.
func TestE2E() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Running Playwright e2e tests...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts")
}

// TestE2EHeaded runs the Playwright e2e suite with a visible browser.
func TestE2EHeaded() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Running Playwright e2e tests (headed)...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts", "--headed")
}

// TestE2EUI launches the Playwright interactive UI runner.
func TestE2EUI() error {
	if err := nodeModulesCheck(); err != nil {
		return err
	}
	fmt.Println("Opening Playwright UI...")
	return sh.RunV("npx", "playwright", "test", "--config", "e2e/playwright.config.ts", "--ui")
}

// TestWatch builds and runs the Go test watcher (cmd/testwatcher) which re-runs tests on .go file changes.
func TestWatch() error {
	fmt.Println("Building and starting Go test watcher...")
	fmt.Println("Tests will run automatically on .go file changes")
	fmt.Println("Press Ctrl+C to stop")

	if err := sh.Run("go", "build", "-o", filepath.Join(binPath, "testwatcher"), "./cmd/testwatcher"); err != nil {
		return fmt.Errorf("failed to build test watcher: %w", err)
	}

	return sh.Run(filepath.Join(binPath, "testwatcher"))
}

// setup:feature:caddy:start

// CaddyInstall installs the Caddy binary into this repo's ./bin directory.
func CaddyInstall() error {
	fmt.Println("Installing Caddy...")
	return installRepoLocalCaddy(".")
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
	caddyBin, err := resolveCaddyBinary()
	if err != nil {
		return err
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

	return sh.Run(caddyBin, "run", "--config", tmpCaddyfile)
}

// setup:feature:caddy:end

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
