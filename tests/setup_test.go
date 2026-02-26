package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"catgoose/go-htmx-demo/internal/setup"

	"github.com/stretchr/testify/require"
)

func TestSetupReplacesAppNameAndModule(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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

	airPath := filepath.Join(dest, ".air.toml")
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
	require.Contains(t, string(gitignoreBytes), "test-app")

	loggerPath := filepath.Join(dest, "internals", "logger", "logger.go")
	loggerBytes, err := os.ReadFile(loggerPath)
	require.NoError(t, err)
	require.Contains(t, string(loggerBytes), `appLogFile = "test-app.log"`)

	_, err = os.Stat(filepath.Join(dest, "Caddyfile"))
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
		if strings.Contains(string(data), "catgoose/go-htmx-demo") {
			t.Errorf("file %s still contains catgoose/go-htmx-demo", rel)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestSetupUsesRandomPortWhenPOmitted(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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

	airPath := filepath.Join(dest, ".air.toml")
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

func assertBuildSucceeds(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(out))
}

// ---------------------------------------------------------------------------
// Feature-combo integration tests
// ---------------------------------------------------------------------------

func TestSetup_FeaturesAll(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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
	assertDirExists(t, filepath.Join(dest, "internals", "ssebroker"))
	assertDirExists(t, filepath.Join(dest, "internals", "database"))
	assertDirExists(t, filepath.Join(dest, "internals", "service", "graph"))
	assertDirExists(t, filepath.Join(dest, "internals", "demo"))
}

func TestSetup_FeaturesNone(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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
	assertDirRemoved(t, filepath.Join(dest, "internals", "ssebroker"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "service", "graph"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "database"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "repository"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "domain"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "demo"))

	_, err = os.Stat(filepath.Join(dest, "Caddyfile"))
	require.True(t, os.IsNotExist(err), "Caddyfile should be removed when no features selected")

	_, err = os.Stat(filepath.Join(dest, "web", "assets", "public", "js", "htmx.ext.sse.js"))
	require.True(t, os.IsNotExist(err), "htmx.ext.sse.js should be removed when sse not selected")
}

func TestSetup_FeaturesAuthOnly(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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
	assertDirRemoved(t, filepath.Join(dest, "internals", "database"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "service", "graph"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "ssebroker"))

	_, err = os.Stat(filepath.Join(dest, "Caddyfile"))
	require.True(t, os.IsNotExist(err), "Caddyfile should be removed when caddy not selected")

	configPath := filepath.Join(dest, "internals", "config", "config.go")
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configContent := string(configBytes)
	require.True(t,
		strings.Contains(configContent, "crooner") || strings.Contains(configContent, "SessionMgr"),
		"config.go should still reference auth-related code (crooner or SessionMgr)",
	)
}

func TestSetup_FeaturesDatabaseOnly(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
	require.NoError(t, err)

	err = setup.Run(context.Background(), dest, setup.Options{
		AppName:    "Database Only App",
		ModulePath: "github.com/test/database-only-app",
		BasePort:   "20400",
		NoCaddy:    false,
		Force:      true,
		Features:   []string{"database"},
	})
	require.NoError(t, err)

	assertNoSetupMarkers(t, dest)
	assertBuildSucceeds(t, dest)
	assertDirRemoved(t, filepath.Join(dest, "internals", "ssebroker"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "service", "graph"))
	assertDirExists(t, filepath.Join(dest, "internals", "database"))
}

func TestSetup_FeaturesSSECaddy(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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
	assertDirExists(t, filepath.Join(dest, "internals", "ssebroker"))

	_, err = os.Stat(filepath.Join(dest, "Caddyfile"))
	require.NoError(t, err, "Caddyfile should exist when caddy is selected")

	assertDirRemoved(t, filepath.Join(dest, "internals", "database"))
	assertDirRemoved(t, filepath.Join(dest, "internals", "service", "graph"))
}

func TestSetup_FeaturesDemo(t *testing.T) {
	repoRoot, err := findRepoRoot()
	require.NoError(t, err)

	dest := t.TempDir()
	err = copyDirExcluding(repoRoot, dest, ".git", "bin", "build", "tmp", "node_modules")
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
	assertDirExists(t, filepath.Join(dest, "internals", "demo"))
}
