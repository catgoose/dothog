package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"catgoose/dothog/internal/setup"

	"github.com/stretchr/testify/require"
)

func TestSetupReplacesAppNameAndModule(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Test App",
		ModulePath: "github.com/example/test-app",
		BasePort:   "12345",
		NoCaddy:    true,
		Force:      true,
	})
	require.NoError(t, err)

	modPath := filepath.Join(dest, "go.mod")
	modBytes, err := os.ReadFile(modPath)
	require.NoError(t, err)
	modContent := string(modBytes)
	require.True(t, strings.HasPrefix(strings.TrimSpace(strings.Split(modContent, "\n")[0]), "module github.com/example/test-app"),
		"go.mod should declare module github.com/example/test-app; got: %s", modContent)

	magePath := filepath.Join(dest, "magefile.go")
	mageBytes, err := os.ReadFile(magePath)
	require.NoError(t, err)
	mageContent := string(mageBytes)
	require.Contains(t, mageContent, `binaryName = "test-app"`)
	require.Contains(t, mageContent, "12345")
	require.Contains(t, mageContent, "12346")
	require.NotContains(t, mageContent, "{{APP_TLS_PORT}}")
	require.NotContains(t, mageContent, "{{TEMPL_HTTP_PORT}}")
	require.NotContains(t, mageContent, "{{CADDY_TLS_PORT}}")

	airPath := filepath.Join(dest, ".air", "server.toml")
	airBytes, err := os.ReadFile(airPath)
	require.NoError(t, err)
	require.Contains(t, string(airBytes), "12346")
	require.NotContains(t, string(airBytes), "{{TEMPL_HTTP_PORT}}")

	readmePath := filepath.Join(dest, "README.md")
	readmeBytes, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	readmeContent := string(readmeBytes)
	require.Contains(t, readmeContent, "Test App")
	require.Contains(t, readmeContent, "12345")
	require.NotContains(t, readmeContent, "{{APP_NAME}}")
	require.NotContains(t, readmeContent, "{{APP_PORT}}")

	envPath := filepath.Join(dest, ".env.development")
	envBytes, err := os.ReadFile(envPath)
	require.NoError(t, err)
	require.Contains(t, string(envBytes), "SERVER_LISTEN_PORT=12345")
	require.NotContains(t, string(envBytes), "{{APP_TLS_PORT}}")
	require.NotContains(t, string(envBytes), "# setup:env")

	gitignorePath := filepath.Join(dest, ".gitignore")
	gitignoreBytes, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	// .gitignore should not contain a bare binary name entry; build/ covers compiled output
	require.NotContains(t, string(gitignoreBytes), "\ntest-app\n",
		".gitignore should not have a bare binary name entry")

	loggerPath := filepath.Join(dest, "internal", "logger", "logger.go")
	loggerBytes, err := os.ReadFile(loggerPath)
	require.NoError(t, err)
	require.Contains(t, string(loggerBytes), `appLogFile = "test-app.log"`)

	_, err = os.Stat(filepath.Join(dest, "config", "Caddyfile"))
	require.True(t, os.IsNotExist(err), "Caddyfile should be removed when using --no-caddy")

	err = filepath.Walk(dest, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			if filepath.Base(path) == "_template_setup" || filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dest, path)
		if strings.HasPrefix(rel, "_template_setup"+string(filepath.Separator)) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(data), "catgoose/dothog") {
			t.Errorf("file %s still contains catgoose/dothog", rel)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestSetupUsesRandomPortWhenPOmitted(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Random Port App",
		ModulePath: "github.com/example/random-port-app",
		BasePort:   "",
		NoCaddy:    true,
		Force:      true,
	})
	require.NoError(t, err)

	envPath := filepath.Join(dest, ".env.development")
	envBytes, err := os.ReadFile(envPath)
	require.NoError(t, err)
	envContent := string(envBytes)
	re := regexp.MustCompile(`SERVER_LISTEN_PORT=(\d+)`)
	matches := re.FindStringSubmatch(envContent)
	require.Len(t, matches, 2, "expected SERVER_LISTEN_PORT in .env.development; got: %s", envContent)
	basePort, err := strconv.Atoi(matches[1])
	require.NoError(t, err)
	require.GreaterOrEqual(t, basePort, 10000, "base port should be >= 10000")
	require.Less(t, basePort, 60000, "base port should be < 60000")

	baseStr := strconv.Itoa(basePort)
	templStr := strconv.Itoa(basePort + 1)

	magePath := filepath.Join(dest, "magefile.go")
	mageBytes, err := os.ReadFile(magePath)
	require.NoError(t, err)
	mageContent := string(mageBytes)
	require.Contains(t, mageContent, baseStr)
	require.Contains(t, mageContent, templStr)
	require.NotContains(t, mageContent, "{{APP_TLS_PORT}}")
	require.NotContains(t, mageContent, "{{TEMPL_HTTP_PORT}}")

	airPath := filepath.Join(dest, ".air", "server.toml")
	airBytes, err := os.ReadFile(airPath)
	require.NoError(t, err)
	require.Contains(t, string(airBytes), templStr)
	require.NotContains(t, string(airBytes), "{{TEMPL_HTTP_PORT}}")
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func copyDirExcluding(src, dst string, excludeDirs ...string) error {
	excludeSet := make(map[string]bool)
	for _, d := range excludeDirs {
		excludeSet[d] = true
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() && excludeSet[filepath.Base(path)] {
			return filepath.SkipDir
		}
		// Skip symlinks — they may point to directories or external paths.
		linfo, lErr := os.Lstat(path)
		if lErr == nil && linfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		destPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}

// ---------------------------------------------------------------------------
// Feature-combo integration test helpers
// ---------------------------------------------------------------------------

func assertNoSetupMarkers(t *testing.T, dir string) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			name := filepath.Base(path)
			if name == "_template_setup" || name == ".git" || name == "node_modules" || name == "tests" || name == "setup" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".templ" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		rel, _ := filepath.Rel(dir, path)
		if strings.Contains(content, "// setup:feature:") {
			t.Errorf("file %s still contains setup:feature marker", rel)
		}
		return nil
	})
	require.NoError(t, err)
}

func assertDirExists(t *testing.T, dir string) {
	t.Helper()
	info, err := os.Stat(dir)
	require.NoError(t, err, "directory should exist: %s", dir)
	require.True(t, info.IsDir())
}

func assertDirRemoved(t *testing.T, dir string) {
	t.Helper()
	_, err := os.Stat(dir)
	require.True(t, os.IsNotExist(err), "directory should not exist: %s", dir)
}

// setupTempDir creates a temp directory and registers a cleanup that removes
// node_modules first. On CI, t.TempDir()'s cleanup can hang for minutes
// deleting thousands of small files from node_modules. Removing it explicitly
// before the framework cleanup prevents test timeout from os.RemoveAll.
func setupTempDir(t *testing.T) string {
	t.Helper()
	dest := t.TempDir()
	t.Cleanup(func() {
		// LIFO: runs before t.TempDir()'s own RemoveAll
		_ = os.RemoveAll(filepath.Join(dest, "node_modules"))
	})
	return dest
}

func assertBuildSucceeds(t *testing.T, dir string) {
	t.Helper()

	// Regenerate templ files after feature stripping removed gated code.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "tool", "templ", "generate")
	cmd.Dir = dir
	cmd.WaitDelay = 10 * time.Second
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "templ generate failed: %s", string(out))

	// Now build.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel2()
	cmd2 := exec.CommandContext(ctx2, "go", "build", "./...")
	cmd2.Dir = dir
	// WaitDelay ensures pipes are closed even if child processes linger
	// after context cancellation kills the parent go build process.
	cmd2.WaitDelay = 10 * time.Second
	out2, err2 := cmd2.CombinedOutput()
	require.NoError(t, err2, "go build failed (timeout=%v): %s", ctx2.Err(), string(out2))
}

// ---------------------------------------------------------------------------
// Feature-combo integration tests
// ---------------------------------------------------------------------------

// TestSetup_NoBareBinaryInGitignore verifies that .gitignore does not contain
// a bare binary name entry after setup. The build/ directory covers compiled output.
func TestSetup_NoBareBinaryInGitignore(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Gitignore Test App",
		ModulePath: "github.com/test/gitignore-test-app",
		BasePort:   "20700",
		Force:      true,
		Features:   setup.AllFeatures,
	})
	require.NoError(t, err)

	gitignoreBytes, err := os.ReadFile(filepath.Join(dest, ".gitignore"))
	require.NoError(t, err)
	content := string(gitignoreBytes)

	// Should not have a bare binary name line (old behavior wrote "dothog" or the new binary name)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		require.NotEqual(t, "gitignore-test-app", trimmed,
			".gitignore should not contain a bare binary name entry")
		require.NotEqual(t, "dothog", trimmed,
			".gitignore should not contain the template binary name")
	}

	// build/ should still be present to cover compiled output
	require.Contains(t, content, "build/")
}

// TestSetup_MageSetupAndInternalSetupRemovable verifies that after setup.Run,
// the target directory can have mage_setup.go and internal/setup removed
// and still build successfully with mage. This simulates the copy-to-new-directory
// flow where these files are removed before the user runs mage.
func TestSetup_MageSetupAndInternalSetupRemovable(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	// Remove setup-only files before running setup (mimics the copy flow in mage_setup.go)
	_ = os.RemoveAll(filepath.Join(dest, "_template_setup"))
	_ = os.RemoveAll(filepath.Join(dest, "internal", "setup"))
	_ = os.Remove(filepath.Join(dest, "mage_setup.go"))

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Mage Clean App",
		ModulePath: "github.com/test/mage-clean-app",
		BasePort:   "20800",
		Force:      true,
		Features:   setup.AllFeatures,
	})
	require.NoError(t, err)

	// mage_setup.go should not exist in the target
	_, err = os.Stat(filepath.Join(dest, "mage_setup.go"))
	require.True(t, os.IsNotExist(err), "mage_setup.go should not exist after removal")

	// internal/setup should not exist in the target
	_, err = os.Stat(filepath.Join(dest, "internal", "setup"))
	require.True(t, os.IsNotExist(err), "internal/setup should not exist after removal")

	// The project should still build without mage_setup.go and internal/setup
	assertBuildSucceeds(t, dest)
}

func TestSetup_FeaturesAll(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	features := make([]string, len(setup.AllFeatures))
	copy(features, setup.AllFeatures)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "All Features App",
		ModulePath: "github.com/test/all-features-app",
		BasePort:   "20100",
		NoCaddy:    false,
		Force:      true,
		Features:   features,
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirExists(t, filepath.Join(dest, "internal", "database"))
	assertDirExists(t, filepath.Join(dest, "internal", "service", "graph"))
	assertDirExists(t, filepath.Join(dest, "internal", "demo"))
}

func TestSetup_FeaturesNone(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "No Features App",
		ModulePath: "github.com/test/no-features-app",
		BasePort:   "20200",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirRemoved(t, filepath.Join(dest, "internal", "service", "graph"))
	assertDirExists(t, filepath.Join(dest, "internal", "database"))          // database is implicit (always kept)
	assertDirRemoved(t, filepath.Join(dest, "internal", "repository"))
	assertDirRemoved(t, filepath.Join(dest, "internal", "domain"))
	assertDirRemoved(t, filepath.Join(dest, "internal", "demo"))

	_, err = os.Stat(filepath.Join(dest, "config", "Caddyfile"))
	require.True(t, os.IsNotExist(err), "Caddyfile should be removed when no features selected")

	_, err = os.Stat(filepath.Join(dest, "web", "assets", "public", "js", "htmx.ext.sse.js"))
	require.True(t, os.IsNotExist(err), "htmx.ext.sse.js should be removed when sse not selected")

	// Dothog-specific docs should be removed (#354)
	for _, f := range []string{"MANIFESTO.md", "AGENTS.md", "README.harmony.md"} {
		_, err = os.Stat(filepath.Join(dest, f))
		require.True(t, os.IsNotExist(err), "%s should be removed during setup", f)
	}

	// scripts/ directory should be removed (#361)
	assertDirRemoved(t, filepath.Join(dest, "scripts"))
	_, err = os.Stat(filepath.Join(dest, ".github", "workflows", "screenshots.yml"))
	require.True(t, os.IsNotExist(err), "screenshots.yml should be removed during setup")

	// db/gen_seed should be removed when demo not selected (#358)
	assertDirRemoved(t, filepath.Join(dest, "db", "gen_seed"))

	// Capacitor files should be removed when capacitor not selected (#353)
	for _, f := range []string{"capacitor.config.ts", "tsconfig.json", "Gemfile"} {
		_, err = os.Stat(filepath.Join(dest, f))
		require.True(t, os.IsNotExist(err), "%s should be removed when capacitor not selected", f)
	}
	assertDirRemoved(t, filepath.Join(dest, "fastlane"))
}

func TestSetup_FeaturesAuthOnly(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Auth Only App",
		ModulePath: "github.com/test/auth-only-app",
		BasePort:   "20300",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{"auth"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirExists(t, filepath.Join(dest, "internal", "database")) // database is implicit
	assertDirRemoved(t, filepath.Join(dest, "internal", "service", "graph"))

	_, err = os.Stat(filepath.Join(dest, "config", "Caddyfile"))
	require.True(t, os.IsNotExist(err), "Caddyfile should be removed when caddy not selected")

	configPath := filepath.Join(dest, "internal", "config", "config.go")
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configContent := string(configBytes)
	require.True(t,
		strings.Contains(configContent, "crooner") || strings.Contains(configContent, "SessionMgr"),
		"config.go should still reference auth-related code (crooner or SessionMgr)",
	)
}

func TestSetup_FeaturesDatabaseOnly(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	// database is implicit — no need to pass it explicitly; MSSQL not selected
	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Database Only App",
		ModulePath: "github.com/test/database-only-app",
		BasePort:   "20400",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirRemoved(t, filepath.Join(dest, "internal", "service", "graph"))
	assertDirExists(t, filepath.Join(dest, "internal", "database"))


	// MSSQL files should be removed
}

func TestSetup_FeaturesMSSQL(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	// database is implicit; explicitly selecting mssql adds MSSQL support
	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "MSSQL App",
		ModulePath: "github.com/test/mssql-app",
		BasePort:   "20450",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{"mssql"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirExists(t, filepath.Join(dest, "internal", "database"))

}

func TestSetup_FeaturesSSECaddy(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "SSE Caddy App",
		ModulePath: "github.com/test/sse-caddy-app",
		BasePort:   "20500",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{"sse", "caddy"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)

	_, err = os.Stat(filepath.Join(dest, "config", "Caddyfile"))
	require.NoError(t, err, "Caddyfile should exist when caddy is selected")

	assertDirExists(t, filepath.Join(dest, "internal", "database")) // database is implicit
	assertDirRemoved(t, filepath.Join(dest, "internal", "service", "graph"))
}

func TestSetup_FeaturesDemo(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Demo App",
		ModulePath: "github.com/test/demo-app",
		BasePort:   "20600",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{"demo", "sse", "caddy"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirExists(t, filepath.Join(dest, "internal", "demo"))

	// db/gen_seed should be kept when demo is selected (#358)
	assertDirExists(t, filepath.Join(dest, "db", "gen_seed"))
}

// TestSetup_PWAWithoutCapacitor verifies that selecting PWA does not pull in
// Capacitor files (#353). PWA is a web feature; Capacitor is a separate opt-in.
func TestSetup_PWAWithoutCapacitor(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "PWA Only App",
		ModulePath: "github.com/test/pwa-only-app",
		BasePort:   "20900",
		Force:      true,
		Features:   []string{"pwa"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)

	// Capacitor files should NOT be present when only PWA is selected
	for _, f := range []string{"capacitor.config.ts", "tsconfig.json", "Gemfile"} {
		_, err = os.Stat(filepath.Join(dest, f))
		require.True(t, os.IsNotExist(err), "%s should not exist with PWA-only setup", f)
	}
	assertDirRemoved(t, filepath.Join(dest, "fastlane"))
	_, err = os.Stat(filepath.Join(dest, ".github", "workflows", "ios.yml"))
	require.True(t, os.IsNotExist(err), "ios.yml should not exist with PWA-only setup")
}

// TestSetup_NoDothogReferences runs setup with all features enabled and verifies
// that no dothog/Dothog/DOTHOG references remain in the derived app. This catches
// template substitution gaps where the original project name leaks through.
func TestSetup_NoDothogReferences(t *testing.T) {
	t.Parallel()
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := setupTempDir(t)
	err = copyDirExcluding(repoRoot, dest, ".git", ".claude", "bin", "build", "log", "tmp", "node_modules")
	require.NoError(t, err)

	// Remove setup-only files (mimics the copy flow in mage_setup.go).
	_ = os.RemoveAll(filepath.Join(dest, "_template_setup"))
	_ = os.RemoveAll(filepath.Join(dest, "internal", "setup"))
	_ = os.Remove(filepath.Join(dest, "mage_setup.go"))
	_ = os.RemoveAll(filepath.Join(dest, "tests"))

	appName := "Test Clean App"
	modulePath := "github.com/example/testcleanapp"

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    appName,
		ModulePath: modulePath,
		BasePort:   "21000",
		NoCaddy:    false,
		Force:      true,
		Features:   setup.AllFeatures,
	})
	require.NoError(t, err)

	// --- Scan all files for residual dothog references ---
	dothogRE := regexp.MustCompile(`(?i)dothog`)

	// Directories and file extensions to skip.
	skipDirs := map[string]bool{
		".git":             true,
		".claude":          true,
		"log":              true,
		"node_modules":     true,
		"_template_setup":  true,
	}

	var violations []string
	err = filepath.Walk(dest, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			if skipDirs[filepath.Base(path)] {
				return filepath.SkipDir
			}
			return nil
		}
		// In a worktree .git is a file, not a directory; skip it.
		if filepath.Base(path) == ".git" {
			return nil
		}
		// Skip binary / non-text files by extension.
		ext := strings.ToLower(filepath.Ext(path))
		binaryExts := map[string]bool{
			".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
			".ico": true, ".woff": true, ".woff2": true, ".ttf": true,
			".eot": true, ".zip": true, ".gz": true, ".tar": true,
			".exe": true, ".dll": true, ".so": true, ".dylib": true,
			".db": true, ".sqlite": true, ".wasm": true,
		}
		if binaryExts[ext] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(dest, path)
		for i, line := range strings.Split(string(data), "\n") {
			if dothogRE.MatchString(line) {
				violations = append(violations, fmt.Sprintf("  %s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	require.NoError(t, err)

	if len(violations) > 0 {
		t.Errorf("found %d residual dothog reference(s) in derived app:\n%s",
			len(violations), strings.Join(violations, "\n"))
	}

	// --- Verify the new app name appears in key files ---

	// go.mod module path
	modBytes, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(strings.TrimSpace(string(modBytes)), "module "+modulePath),
		"go.mod should declare module %s", modulePath)

	// HTML title in the public index.html
	indexPath := filepath.Join(dest, "web", "assets", "public", "index.html")
	if indexBytes, readErr := os.ReadFile(indexPath); readErr == nil {
		require.Contains(t, string(indexBytes), appName,
			"index.html should contain the app name")
	}

	// --- Confirm the derived app compiles ---
	assertBuildSucceeds(t, dest)
}
