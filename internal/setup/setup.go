package setup

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Template identity used by setup to detect and replace module paths.
const (
	TemplateModule   = "catgoose/dothog"
	TemplateSetupDir = "_template_setup"
)

// templateName is the last path segment of TemplateModule (e.g. "dothog" or "harmony").
// Used as the default binary/app name that setup replaces with the user's chosen name.
var templateName = filepath.Base(TemplateModule)

// setupEnvPrefix marks comment lines in .env.development that carry template
// variables for setup.  During setup the prefix is stripped (activating the
// template line) and the following literal-value line is removed.
const setupEnvPrefix = "# setup:env "

// Feature tags for the interactive feature selector.
const (
	FeatureAuth     = "auth"
	FeatureGraph    = "graph"
	FeatureDatabase = "database"
	FeatureMSSQL    = "mssql"
	FeaturePostgres = "postgres"
	FeatureSSE      = "sse"
	FeatureCaddy    = "caddy"
	FeatureAvatar   = "avatar"
	FeatureDemo             = "demo"
	FeatureSessionSettings  = "session_settings"
	FeatureAlpine           = "alpine"
	FeatureCapacitor        = "capacitor"
	FeatureOffline          = "offline"
	FeatureSync             = "sync"
	FeatureCSRF             = "csrf"
	FeatureLinkRelations    = "link_relations"
	FeatureWebStandards     = "web_standards"
	FeatureBrowserAPIs      = "browser_apis"
	FeaturePWA              = "pwa"
)

// AllFeatures lists every selectable feature tag.
// "database" is always included (implied by the base template) and is not user-selectable.
var AllFeatures = []string{FeatureAuth, FeatureGraph, FeatureDatabase, FeatureMSSQL, FeaturePostgres, FeatureSSE, FeatureCaddy, FeatureAvatar, FeatureDemo, FeatureSessionSettings, FeatureAlpine, FeatureCapacitor, FeatureOffline, FeatureSync, FeatureCSRF, FeatureLinkRelations, FeatureWebStandards, FeatureBrowserAPIs, FeaturePWA}

// ImplicitFeatures are always selected and not presented to the user.
// "database" is implicit because SQLite is the base database engine.
// "alpine" is implicit because Alpine.js is the standard client-side state layer.
var ImplicitFeatures = []string{FeatureDatabase, FeatureAlpine}

// featureDeps maps a feature to the features it implies.
// pwa -> sync -> offline.  capacitor is a separate opt-in for native wrapping.
var featureDeps = map[string][]string{
	FeatureSync:           {FeatureOffline},
	FeaturePWA:            {FeatureOffline, FeatureSync},
	FeatureDemo:            {FeatureSessionSettings},
	FeatureCSRF:           {FeatureSessionSettings},
	FeatureLinkRelations:  {FeatureSessionSettings},
	FeatureBrowserAPIs:    {FeatureSSE},
	FeatureWebStandards:   {FeatureSessionSettings},
}

// ExpandFeatureDeps adds any transitive dependencies implied by the
// selected features. For example, selecting "sync" pulls in "offline"
// and "capacitor".
func ExpandFeatureDeps(features []string) []string {
	have := make(map[string]bool, len(features))
	for _, f := range features {
		have[f] = true
	}
	changed := true
	for changed {
		changed = false
		for f := range have {
			for _, dep := range featureDeps[f] {
				if !have[dep] {
					have[dep] = true
					changed = true
				}
			}
		}
	}
	out := make([]string, 0, len(have))
	for _, f := range AllFeatures {
		if have[f] {
			out = append(out, f)
		}
	}
	// Preserve any features not in AllFeatures (shouldn't happen, but safe).
	for f := range have {
		found := false
		for _, af := range AllFeatures {
			if af == f {
				found = true
				break
			}
		}
		if !found {
			out = append(out, f)
		}
	}
	return out
}

// Options configures the template setup run.
type Options struct {
	AppName    string
	ModulePath string
	BasePort   string
	Features   []string
	Force      bool
	NoCaddy    bool

	// ConfirmFunc is an optional interactive confirm callback.  When non-nil
	// it is used to prompt the user (e.g. for certificate generation).  When
	// nil, any prompt-gated action is executed silently.
	ConfirmFunc func(msg string) (bool, error)
}

func binaryNameFromApp(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, " ", "-")
}

func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	return "", scanner.Err()
}

func replaceInFile(path string, old, new string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ReplaceAll(string(data), old, new)
	return os.WriteFile(path, []byte(content), 0644)
}

// replaceTemplateNames walks all text files under dir and replaces standalone
// template name references (e.g. dothog/Dothog/DOTHOG) with the derived app's
// binary name and app name. This runs after the module-path replacement
// (catgoose/dothog → new module) so only standalone references remain.
func replaceTemplateNames(dir, binaryName, appName string) error {
	upperName := strings.ToUpper(binaryName)
	titleTemplateName := strings.ToUpper(templateName[:1]) + templateName[1:]
	upperTemplateName := strings.ToUpper(templateName)

	return filepath.Walk(dir, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".claude" || name == TemplateSetupDir || name == "log" || name == "node_modules" || name == "tests" {
				return filepath.SkipDir
			}
			// Skip setup package — it contains the template name in
			// replacement logic and function names that must not be rewritten.
			rel, _ := filepath.Rel(dir, path)
			if rel == filepath.Join("internal", "setup") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		// Skip go.sum — checksums are regenerated by go mod tidy.
		if rel == "go.sum" {
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if !strings.Contains(content, templateName) && !strings.Contains(content, titleTemplateName) && !strings.Contains(content, upperTemplateName) {
			return nil
		}
		content = strings.ReplaceAll(content, upperTemplateName, upperName)
		content = strings.ReplaceAll(content, titleTemplateName, appName)
		content = strings.ReplaceAll(content, templateName, binaryName)
		return os.WriteFile(path, []byte(content), info.Mode())
	})
}

// Run performs template setup: module path, ports, app name, and related file updates.
func Run(ctx context.Context, dir string, opts Options) error {
	if opts.AppName == "" {
		return fmt.Errorf("APP_NAME is required")
	}
	binaryName := binaryNameFromApp(opts.AppName)

	currentModule, err := readModulePath(dir)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}
	if currentModule != TemplateModule && !opts.Force {
		return fmt.Errorf("module path is already customized (go.mod: %s); pass Force to run setup again", currentModule)
	}

	modulePath := opts.ModulePath
	if modulePath == "" {
		if currentModule == TemplateModule {
			modulePath = TemplateModule + "-" + binaryName
		} else {
			modulePath = currentModule
		}
	}

	basePort := opts.BasePort
	if basePort == "" {
		basePort = strconv.Itoa(10000 + rand.Intn(50000))
	}
	if len(basePort) != 5 {
		return fmt.Errorf("BASE_PORT must be a 5-digit number, got: %s", basePort)
	}
	baseNum, err := strconv.Atoi(basePort)
	if err != nil || baseNum >= 60000 {
		return fmt.Errorf("BASE_PORT must be < 60000, got: %s", basePort)
	}
	appTLSPort := basePort
	templHTTPPort := strconv.Itoa(baseNum + 1)
	caddyTLSPort := strconv.Itoa(baseNum + 2)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	gomodPath := filepath.Join(dir, "go.mod")
	if err := replaceInFile(gomodPath, "module "+TemplateModule, "module "+modulePath); err != nil {
		return fmt.Errorf("updating go.mod: %w", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == TemplateSetupDir {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, ".git"+string(filepath.Separator)) || strings.HasPrefix(rel, TemplateSetupDir+string(filepath.Separator)) {
			return nil
		}
		// go.mod was already updated above; skip to avoid double replacement
		if rel == "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !strings.Contains(string(data), TemplateModule) {
			return nil
		}
		content := strings.ReplaceAll(string(data), TemplateModule, modulePath)
		return os.WriteFile(path, []byte(content), 0644)
	})
	if err != nil {
		return fmt.Errorf("replacing module path: %w", err)
	}

	binaryNameRe := regexp.MustCompile(`binaryName\s*=\s*"` + regexp.QuoteMeta(templateName) + `"`)
	magePath := filepath.Join(dir, "magefile.go")
	mageData, err := os.ReadFile(magePath)
	if err != nil {
		return fmt.Errorf("reading magefile: %w", err)
	}
	mageContent := string(mageData)
	mageContent = binaryNameRe.ReplaceAllString(mageContent, `binaryName = "`+binaryName+`"`)
	mageContent = strings.ReplaceAll(mageContent, "{{APP_TLS_PORT}}", appTLSPort)
	mageContent = strings.ReplaceAll(mageContent, "{{TEMPL_HTTP_PORT}}", templHTTPPort)
	mageContent = strings.ReplaceAll(mageContent, "{{CADDY_TLS_PORT}}", caddyTLSPort)
	if err := os.WriteFile(magePath, []byte(mageContent), 0644); err != nil {
		return err
	}

	for _, f := range []string{filepath.Join("config", "Caddyfile"), filepath.Join(".air", "server.toml")} {
		p := filepath.Join(dir, f)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		content = strings.ReplaceAll(content, "{{APP_TLS_PORT}}", appTLSPort)
		content = strings.ReplaceAll(content, "{{TEMPL_HTTP_PORT}}", templHTTPPort)
		content = strings.ReplaceAll(content, "{{CADDY_TLS_PORT}}", caddyTLSPort)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Legacy --no-caddy flag: remove Caddyfile if requested.
	// The new Features mechanism handles this too (see removeOptionalContent).
	if opts.NoCaddy {
		_ = os.Remove(filepath.Join(dir, "config", "Caddyfile"))
	}

	// Compose .env.dev and .env.development from the tracked .env.development.
	// Lines tagged with "# setup:env " are activated (prefix stripped, literal
	// default removed) and template placeholders are resolved to real ports.
	envDevPath := filepath.Join(dir, ".env.development")
	if data, err := os.ReadFile(envDevPath); err == nil {
		content := composeSetupEnv(string(data))
		content = strings.ReplaceAll(content, "{{APP_TLS_PORT}}", appTLSPort)
		content = strings.ReplaceAll(content, "{{TEMPL_HTTP_PORT}}", templHTTPPort)
		content = strings.ReplaceAll(content, "{{CADDY_TLS_PORT}}", caddyTLSPort)
		// Ensure APP_NAME is set in the generated env file
		if !strings.Contains(content, "APP_NAME=") {
			content += "\n# Application\nAPP_NAME=" + opts.AppName + "\n"
		}
		for _, envTarget := range []string{".env.dev", ".env.development"} {
			if err := os.WriteFile(filepath.Join(dir, envTarget), []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	// Rewrite Dockerfile: binary name and port
	dockerfilePath := filepath.Join(dir, "Dockerfile")
	if data, err := os.ReadFile(dockerfilePath); err == nil {
		content := string(data)
		content = strings.ReplaceAll(content, "-o /"+templateName, "-o /"+binaryName)
		content = strings.ReplaceAll(content, "build /"+templateName, "build /"+binaryName)
		content = strings.ReplaceAll(content, "/usr/local/bin/"+templateName, "/usr/local/bin/"+binaryName)
		content = strings.ReplaceAll(content, `ENTRYPOINT ["`+templateName+`"]`, `ENTRYPOINT ["`+binaryName+`"]`)
		content = strings.ReplaceAll(content, "SERVER_LISTEN_PORT=3000", "SERVER_LISTEN_PORT="+appTLSPort)
		content = strings.ReplaceAll(content, "EXPOSE 3000", "EXPOSE "+appTLSPort)
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	if data, err := os.ReadFile(filepath.Join(dir, "package-lock.json")); err == nil {
		content := strings.ReplaceAll(string(data), `"name": "`+templateName+`"`, `"name": "`+binaryName+`"`)
		content = regexp.MustCompile(`"name":\s*"` + regexp.QuoteMeta(templateName) + `"`).ReplaceAllString(content, `"name": "`+binaryName+`"`)
		if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(content), 0644); err != nil {
			return err
		}
	}



	loggerPath := filepath.Join(dir, "internal", "logger", "logger.go")
	if data, err := os.ReadFile(loggerPath); err == nil {
		content := strings.ReplaceAll(string(data), `appLogFile = "`+templateName+`.log"`, `appLogFile = "`+binaryName+`.log"`)
		if err := os.WriteFile(loggerPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Replace {{BINARY_NAME}} placeholder in JS and Go source files so that
	// derived apps get unique cache names, IndexedDB databases, cookie names,
	// and BroadcastChannel identifiers.
	for _, f := range []string{
		filepath.Join("web", "assets", "public", "js", "sw.js"),
		filepath.Join("web", "assets", "public", "js", "sync.js"),
		filepath.Join("web", "assets", "public", "js", "broadcast.js"),
		filepath.Join("internal", "routes", "middleware", "session_settings.go"),
	} {
		p := filepath.Join(dir, f)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := strings.ReplaceAll(string(data), "{{BINARY_NAME}}", binaryName)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			return err
		}
	}

	_ = os.Remove(filepath.Join(dir, ".golangci.yml"))

	// Broad replacement of standalone template name references that survive
	// the module-path replacement above (app names, binary names, HTML
	// titles, test assertions, doc references, CI workflows, etc.).
	if err := replaceTemplateNames(dir, binaryName, opts.AppName); err != nil {
		return fmt.Errorf("replacing template names: %w", err)
	}

	// Generate the derived app's README from the template.  This runs after
	// replaceTemplateNames so that the intentional "generated from
	// catgoose/dothog" reference is not rewritten to the derived app's name.
	templateReadme := filepath.Join(dir, TemplateSetupDir, "README.template.md")
	if data, err := os.ReadFile(templateReadme); err == nil {
		content := string(data)
		content = strings.ReplaceAll(content, "{{APP_TLS_PORT}}", appTLSPort)
		content = strings.ReplaceAll(content, "{{TEMPL_HTTP_PORT}}", templHTTPPort)
		content = strings.ReplaceAll(content, "{{CADDY_TLS_PORT}}", caddyTLSPort)
		content = strings.ReplaceAll(content, "{{APP_NAME}}", opts.AppName)
		content = strings.ReplaceAll(content, "{{BINARY_NAME}}", binaryName)
		content = strings.ReplaceAll(content, "{{MODULE_PATH}}", modulePath)
		content = strings.ReplaceAll(content, "{{TEMPLATE_REF}}", "["+TemplateModule+"](https://github.com/"+TemplateModule+")")
		content = strings.ReplaceAll(content, "{{FEATURE_TABLE}}", buildFeatureTable(opts.Features))
		content = strings.ReplaceAll(content, "{{FEATURE_SECTIONS}}", buildFeatureSections(opts.Features))
		content = strings.ReplaceAll(content, "{{TECH_STACK}}", buildTechStack(opts.Features))
		content = strings.ReplaceAll(content, "{{QUICK_START}}", buildQuickStart(binaryName, appTLSPort))
		content = strings.ReplaceAll(content, "{{ENV_TABLE}}", buildEnvTable(opts.Features, opts.AppName, appTLSPort))
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	if err := removeOptionalContent(dir, opts); err != nil {
		return fmt.Errorf("removing optional content: %w", err)
	}

	if err := ensureCerts(ctx, absDir, opts); err != nil {
		return fmt.Errorf("ensuring certificates: %w", err)
	}

	if err := cleanOrphanedImports(dir, modulePath); err != nil {
		return fmt.Errorf("cleaning orphaned imports: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = absDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Failsafe: install npm dependencies if node_modules is missing
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(dir, "package-lock.json")); err == nil {
			npmCmd := exec.CommandContext(ctx, "npm", "ci")
			npmCmd.Dir = absDir
			npmCmd.Stdout = os.Stdout
			npmCmd.Stderr = os.Stderr
			if err := npmCmd.Run(); err != nil {
				return fmt.Errorf("npm ci: %w", err)
			}
		}
	}

	gitDir := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
		cmd.Dir = absDir
		if err := cmd.Run(); err != nil {
			cmdAdd := exec.CommandContext(ctx, "git", "remote", "add", "origin", "https://"+modulePath+".git")
			cmdAdd.Dir = absDir
			_ = cmdAdd.Run()
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Marker constants
// ---------------------------------------------------------------------------

// featureFileMarkerPrefix matches "// setup:feature:TAG" as the first non-blank
// line to mark an entire file for removal when TAG is deselected.
const featureFileMarkerPrefix = "// setup:feature:"

// featureBlockStartPrefix matches "// setup:feature:TAG:start"
const featureBlockStartPrefix = "// setup:feature:"

// featureBlockStartSuffix is appended to the tag to build the start marker.
const featureBlockStartSuffix = ":start"

// featureBlockEndSuffix is appended to the tag to build the end marker.
const featureBlockEndSuffix = ":end"

// ---------------------------------------------------------------------------
// Feature-aware content removal
// ---------------------------------------------------------------------------

// removeOptionalContent strips deselected feature content (when opts.Features
// is non-nil). Demo content is removed unless the "demo" feature is selected.
func removeOptionalContent(dir string, opts Options) error {
	// Build set of features being removed (empty when Features is nil — legacy mode)
	removeTags := make(map[string]bool)
	if opts.Features != nil {
		expanded := ExpandFeatureDeps(opts.Features)
		keep := make(map[string]bool)
		for _, f := range expanded {
			keep[f] = true
		}
		// Implicit features are always kept
		for _, f := range ImplicitFeatures {
			keep[f] = true
		}
		for _, f := range AllFeatures {
			if !keep[f] {
				removeTags[f] = true
			}
		}
	} else {
		// Legacy/programmatic usage (Features == nil): remove demo content
		removeTags[FeatureDemo] = true
	}

	// Remove dothog-specific documentation that is not relevant to derived apps.
	_ = os.Remove(filepath.Join(dir, "MANIFESTO.md"))
	_ = os.Remove(filepath.Join(dir, "AGENTS.md"))
	_ = os.Remove(filepath.Join(dir, "README.harmony.md"))

	// Remove dothog-specific scripts (doc generation, screenshot automation).
	_ = os.RemoveAll(filepath.Join(dir, "scripts"))
	_ = os.Remove(filepath.Join(dir, ".github", "workflows", "screenshots.yml"))
	_ = os.Remove(filepath.Join(dir, ".github", "workflows", "docs.yml"))
	_ = os.Remove(filepath.Join(dir, ".github", "workflows", "pipeline.yml"))
	_ = os.RemoveAll(filepath.Join(dir, ".github", "harmony"))

	// Remove the entire docs/ directory — it contains only dothog-specific
	// documentation (ARCHITECTURE.md, HAL.md, etc.) that isn't useful in
	// derived apps.  The generated README is sufficient.
	_ = os.RemoveAll(filepath.Join(dir, "docs"))

	// Remove the setup package itself — it only exists for template setup (#377).
	_ = os.RemoveAll(filepath.Join(dir, "internal", "setup"))
	_ = os.Remove(filepath.Join(dir, "tests", "setup_test.go"))

	// Replace demo-specific e2e tests with a minimal smoke suite (#356).
	// Keep helpers.ts and playwright.config.ts (they contain general utilities
	// and the binary-name aware config). Remove all demo page spec files and
	// generate a smoke test that verifies the app loads.
	replaceE2EWithSmoke(dir, opts.AppName)

	// Remove all .db files (and WAL/SHM) so derived apps start clean.
	// The app auto-creates databases on first start via os.MkdirAll + EnsureSchema.
	if matches, err := filepath.Glob(filepath.Join(dir, "db", "*.db*")); err == nil {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}

	// Seed data generation is only relevant when the demo feature is selected.
	if removeTags[FeatureDemo] {
		_ = os.RemoveAll(filepath.Join(dir, "db", "gen_seed"))
	}

	if removeTags[FeatureSSE] {
		_ = os.Remove(filepath.Join(dir, "web", "assets", "public", "js", "htmx.ext.sse.js"))
	}
	if removeTags[FeatureCaddy] {
		_ = os.Remove(filepath.Join(dir, "config", "Caddyfile"))
	}
	if removeTags[FeatureCapacitor] {
		_ = os.Remove(filepath.Join(dir, "capacitor.config.ts"))
		_ = os.Remove(filepath.Join(dir, "tsconfig.json"))
		_ = os.RemoveAll(filepath.Join(dir, "fastlane"))
		_ = os.Remove(filepath.Join(dir, "Gemfile"))
		_ = os.Remove(filepath.Join(dir, ".github", "workflows", "ios.yml"))
	}
	// Alpine.js is always included (implicit feature); no removal needed.

	// Favicon handling: demo uses hot dog, non-demo uses generic defaults.
	if removeTags[FeatureDemo] {
		replaceDefaultFavicons(dir)
	} else {
		_ = os.RemoveAll(filepath.Join(dir, "web", "assets", "public", "images", "default"))
	}

	var toRemove []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == TemplateSetupDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)

		// Check whole-file markers
		modified := false
		if tag := featureFileTag(content); tag != "" {
			if removeTags[tag] {
				toRemove = append(toRemove, path)
				if strings.HasSuffix(path, ".templ") {
					toRemove = append(toRemove, strings.TrimSuffix(path, ".templ")+"_templ.go")
				}
				return nil
			}
			// Feature is selected — strip the marker line but keep the file.
			content = stripFeatureFileMarker(content)
			modified = true
			// Also strip the marker from the generated _templ.go companion
			// where it appears but not as the first line.
			if strings.HasSuffix(path, ".templ") {
				templGoPath := strings.TrimSuffix(path, ".templ") + "_templ.go"
				if tgData, err := os.ReadFile(templGoPath); err == nil {
					tgContent := stripFeatureFileMarkerAnywhere(string(tgData), tag)
					if tgContent != string(tgData) {
						_ = os.WriteFile(templGoPath, []byte(tgContent), info.Mode())
					}
				}
			}
		}

		// Strip blocks
		cleaned := stripBlocks(content, removeTags)
		if cleaned != content {
			modified = true
		}
		if modified {
			return os.WriteFile(path, []byte(cleaned), info.Mode())
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning for markers: %w", err)
	}

	for _, path := range toRemove {
		_ = os.Remove(path)
	}

	return removeEmptyDirs(dir)
}

// featureFileTag returns the feature tag if the first non-blank line is
// "// setup:feature:TAG" (without :start/:end suffix). Returns "" otherwise.
func featureFileTag(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, featureFileMarkerPrefix) {
			rest := strings.TrimPrefix(line, featureFileMarkerPrefix)
			// Must be a plain tag (no :start or :end suffix)
			if !strings.Contains(rest, ":") {
				return rest
			}
		}
		return ""
	}
	return ""
}

// stripFeatureFileMarker removes the "// setup:feature:TAG" line (the first
// non-blank line) from content, preserving everything else. It only strips
// whole-file markers (no :start/:end suffix).
func stripFeatureFileMarker(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	stripped := false
	for _, line := range lines {
		if !stripped {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				out = append(out, line)
				continue
			}
			if strings.HasPrefix(trimmed, featureFileMarkerPrefix) {
				rest := strings.TrimPrefix(trimmed, featureFileMarkerPrefix)
				if !strings.Contains(rest, ":") {
					stripped = true
					continue // skip the marker line
				}
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// stripFeatureFileMarkerAnywhere removes the "// setup:feature:TAG" line
// anywhere in the file (not just the first non-blank line). This handles
// generated _templ.go files where the marker is after the templ header.
func stripFeatureFileMarkerAnywhere(content, tag string) string {
	marker := featureFileMarkerPrefix + tag
	lines := strings.Split(content, "\n")
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == marker {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// stripBlocks removes feature-tagged blocks for deselected features. Supports
// nesting: an inner block is removed if its own tag OR any enclosing block's
// tag is being removed.
func stripBlocks(content string, removeTags map[string]bool) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Feature block start: // setup:feature:TAG:start
		if tag, ok := parseFeatureBlockStart(trimmed); ok {
			if skipDepth > 0 || removeTags[tag] {
				skipDepth++
			}
			continue // always strip the marker line itself
		}

		// Feature block end: // setup:feature:TAG:end
		if _, ok := parseFeatureBlockEnd(trimmed); ok {
			if skipDepth > 0 {
				skipDepth--
			}
			continue // always strip the marker line itself
		}

		if skipDepth == 0 {
			out = append(out, line)
		}
	}

	return collapseBlankLines(out)
}

// parseFeatureBlockStart checks for "// setup:feature:TAG:start" and returns the tag.
func parseFeatureBlockStart(trimmed string) (string, bool) {
	if !strings.HasPrefix(trimmed, featureBlockStartPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(trimmed, featureBlockStartPrefix)
	if strings.HasSuffix(rest, featureBlockStartSuffix) {
		tag := strings.TrimSuffix(rest, featureBlockStartSuffix)
		if tag != "" {
			return tag, true
		}
	}
	return "", false
}

// parseFeatureBlockEnd checks for "// setup:feature:TAG:end" and returns the tag.
func parseFeatureBlockEnd(trimmed string) (string, bool) {
	if !strings.HasPrefix(trimmed, featureBlockStartPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(trimmed, featureBlockStartPrefix)
	if strings.HasSuffix(rest, featureBlockEndSuffix) {
		tag := strings.TrimSuffix(rest, featureBlockEndSuffix)
		if tag != "" {
			return tag, true
		}
	}
	return "", false
}

// collapseBlankLines collapses runs of 3+ blank lines down to 2.
func collapseBlankLines(lines []string) string {
	var collapsed []string
	blanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks > 2 {
				continue
			}
		} else {
			blanks = 0
		}
		collapsed = append(collapsed, line)
	}
	return strings.Join(collapsed, "\n")
}

// ---------------------------------------------------------------------------
// Orphaned import cleanup
// ---------------------------------------------------------------------------

// importLineRe matches a single Go import line: optional alias + quoted path.
var importLineRe = regexp.MustCompile(`^\s*(?:(\w+)\s+)?"([^"]+)"`)

// cleanOrphanedImports walks all .go files under dir. For each import of an
// internal package (starts with modulePath), it checks whether the package
// directory still exists. If not, it removes the import line. It also removes
// stdlib imports whose package name is no longer referenced in the file body.
func cleanOrphanedImports(dir, modulePath string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == TemplateSetupDir || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		cleaned := removeOrphanedImportLines(content, dir, modulePath)
		if cleaned != content {
			return os.WriteFile(path, []byte(cleaned), info.Mode())
		}
		return nil
	})
}

// removeOrphanedImportLines scans a Go source file and removes import lines
// that reference internal packages whose directories no longer exist, plus
// stdlib imports whose package identifier is no longer used in the file body.
func removeOrphanedImportLines(content, baseDir, modulePath string) string {
	lines := strings.Split(content, "\n")

	// First pass: collect import info
	type importInfo struct {
		alias      string
		importPath string
		pkgName    string
		lineIdx    int
	}
	var imports []importInfo
	inImportBlock := false
	importStart := -1
	importEnd := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import (" {
			inImportBlock = true
			importStart = i
			continue
		}
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			importEnd = i
			continue
		}
		if inImportBlock {
			if m := importLineRe.FindStringSubmatch(trimmed); m != nil {
				alias := m[1]
				impPath := m[2]
				pkgName := goPkgName(impPath)
				imports = append(imports, importInfo{
					lineIdx:    i,
					alias:      alias,
					importPath: impPath,
					pkgName:    pkgName,
				})
			}
		}
	}

	if importStart < 0 || importEnd < 0 {
		return content
	}

	// Build the file body (everything outside the import block)
	var bodyParts []string
	for i, line := range lines {
		if i >= importStart && i <= importEnd {
			continue
		}
		bodyParts = append(bodyParts, line)
	}
	body := strings.Join(bodyParts, "\n")

	// Determine which import lines to remove
	removeLines := make(map[int]bool)
	for _, imp := range imports {
		// Check internal packages: directory must exist AND identifier must be used.
		if strings.HasPrefix(imp.importPath, modulePath+"/") {
			relPkg := strings.TrimPrefix(imp.importPath, modulePath+"/")
			pkgDir := filepath.Join(baseDir, filepath.FromSlash(relPkg))
			if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
				removeLines[imp.lineIdx] = true
				continue
			}
			// Directory exists but identifier may be unused after feature stripping.
			// Check if the package identifier appears as "ident." (a qualified reference)
			// to avoid false positives from the word appearing in strings/comments.
			ident := imp.pkgName
			if imp.alias != "" && imp.alias != "_" {
				ident = imp.alias
			}
			if imp.alias == "_" {
				continue
			}
			qualifiedRe := regexp.MustCompile(regexp.QuoteMeta(ident) + `\.`)
			if !qualifiedRe.MatchString(body) {
				removeLines[imp.lineIdx] = true
			}
			continue
		}

		// For stdlib imports (no dot in first path segment), check if the
		// identifier is still referenced in the file body. External packages
		// (github.com/*, gopkg.in/*, etc.) are left alone — go mod tidy
		// handles those, and their package names can't be reliably inferred
		// from import paths.
		firstSeg := strings.SplitN(imp.importPath, "/", 2)[0]
		if strings.Contains(firstSeg, ".") {
			continue // external package — skip body-reference check
		}
		ident := imp.pkgName
		if imp.alias != "" && imp.alias != "_" {
			ident = imp.alias
		}
		if imp.alias == "_" {
			continue // side-effect imports are always kept
		}
		// Use word-boundary check: the identifier must appear as a standalone word
		identRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(ident) + `\b`)
		if !identRe.MatchString(body) {
			removeLines[imp.lineIdx] = true
		}
	}

	if len(removeLines) == 0 {
		return content
	}

	var out []string
	for i, line := range lines {
		if removeLines[i] {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// ---------------------------------------------------------------------------
// Certificate generation
// ---------------------------------------------------------------------------

// ensureCerts checks whether localhost certificates already exist in dir.
// If found, they are kept as-is.  If missing and the caddy feature is
// selected, the user is prompted (via opts.ConfirmFunc) to generate new
// self-signed certificates.  When ConfirmFunc is nil (programmatic / test
// usage) generation happens silently.
func ensureCerts(ctx context.Context, dir string, opts Options) error {
	certPath := filepath.Join(dir, "localhost.crt")
	keyPath := filepath.Join(dir, "localhost.key")

	// If both files already exist locally, use them.
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			fmt.Println("Using existing localhost certificates.")
			return nil
		}
	}

	// Only generate certificates when caddy is selected.
	if !hasCaddyFeature(opts) {
		return nil
	}

	// Prompt the user when an interactive confirm is available.
	if opts.ConfirmFunc != nil {
		generate, err := opts.ConfirmFunc("No localhost certificates found. Generate self-signed certificates?")
		if err != nil {
			return err
		}
		if !generate {
			fmt.Println("Skipping certificate generation.")
			fmt.Println("You will need to provide localhost.crt and localhost.key for HTTPS to work.")
			return nil
		}
	}

	fmt.Println("Generating self-signed localhost certificates...")
	if err := generateCerts(ctx, dir); err != nil {
		return err
	}
	fmt.Println("NOTE: You will need to install these certificates in your system trust store.")
	fmt.Println("See the HTTPS Development Setup section in README.md for instructions.")
	return nil
}

// hasCaddyFeature reports whether the caddy feature is selected.
func hasCaddyFeature(opts Options) bool {
	if opts.Features == nil {
		return !opts.NoCaddy // legacy mode: caddy unless --no-caddy
	}
	for _, f := range opts.Features {
		if f == FeatureCaddy {
			return true
		}
	}
	return false
}

// generateCerts creates a self-signed localhost TLS certificate and key in dir
// using openssl. The generated files are localhost.crt and localhost.key.
func generateCerts(ctx context.Context, dir string) error {
	certPath := filepath.Join(dir, "localhost.crt")
	keyPath := filepath.Join(dir, "localhost.key")

	cmd := exec.CommandContext(ctx, "openssl", "req",
		"-x509",
		"-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "365",
		"-nodes",
		"-subj", "/CN=localhost",
		"-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1",
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("openssl: %w", err)
	}
	fmt.Printf("Generated development certificates:\n  %s\n  %s\n", certPath, keyPath)
	return nil
}

// versionSuffixRe matches Go major-version path segments like "v2", "v3", etc.
var versionSuffixRe = regexp.MustCompile(`^v\d+$`)

// gopkgVersionRe matches gopkg.in version suffixes like ".v2", ".v3"
var gopkgVersionRe = regexp.MustCompile(`\.\w+$`)

// goPkgName returns the Go package identifier for an import path.
// For versioned modules (e.g. "github.com/foo/bar/v4") it returns
// the second-to-last segment ("bar"). For gopkg.in paths like
// "gopkg.in/foo/bar.v2" it strips the version suffix ("bar").
func goPkgName(importPath string) string {
	parts := strings.Split(importPath, "/")
	last := parts[len(parts)-1]
	// Handle major-version suffixes: "echo/v4" → "echo"
	if versionSuffixRe.MatchString(last) && len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	// Handle gopkg.in version suffixes: "lumberjack.v2" → "lumberjack"
	if strings.HasPrefix(importPath, "gopkg.in/") {
		return gopkgVersionRe.ReplaceAllString(last, "")
	}
	return last
}

// ---------------------------------------------------------------------------
// README feature generation (#360)
// ---------------------------------------------------------------------------

// featureDescriptions maps feature tags to human-readable descriptions
// for the generated README.
var featureDescriptions = map[string]struct{ label, desc string }{
	FeatureAuth:            {"Auth (Crooner)", "Azure AD / OIDC authentication via crooner"},
	FeatureGraph:           {"Graph API", "Microsoft Graph API integration for user data"},
	FeatureAvatar:          {"Avatar Photos", "User profile photo download and caching from Graph"},
	FeatureDatabase:        {"Database (fraggle)", "Repository layer with schema DSL (SQLite base)"},
	FeatureMSSQL:           {"MSSQL", "Microsoft SQL Server dialect support"},
	FeaturePostgres:        {"PostgreSQL", "PostgreSQL dialect support"},
	FeatureSSE:             {"SSE", "Server-Sent Events with HTMX integration"},
	FeatureCaddy:           {"Caddy (HTTPS)", "Caddy reverse proxy with TLS termination"},
	FeatureDemo:            {"Demo Content", "Demo pages, seed data, and example routes"},
	FeatureSessionSettings: {"Session Settings", "Per-session theme and layout preferences"},
	FeatureAlpine:          {"Alpine.js", "Client-side state management"},
	FeatureCapacitor:       {"Capacitor", "Native mobile wrapper (iOS/Android)"},
	FeatureOffline:         {"Offline Mode", "Service worker and write queue for offline use"},
	FeatureSync:            {"Sync", "SQLite data synchronization between client and server"},
	FeatureCSRF:            {"CSRF Protection", "Token-based CSRF with optional per-request rotation"},
	FeatureLinkRelations:   {"Link Relations", "Context bars, breadcrumbs, and site map"},
	FeatureWebStandards:    {"Web Standards", "Server-Timing, Vary, Permissions-Policy, Early Hints"},
	FeatureBrowserAPIs:     {"Browser APIs", "sendBeacon, BroadcastChannel integration"},
	FeaturePWA:             {"PWA", "Progressive Web App with offline + sync support"},
}

// buildFeatureTable generates a markdown table of enabled features for the README.
func buildFeatureTable(features []string) string {
	if features == nil {
		return "_No feature configuration provided._"
	}

	expanded := ExpandFeatureDeps(features)
	keep := make(map[string]bool)
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}

	var sb strings.Builder
	sb.WriteString("| Feature | Description |\n")
	sb.WriteString("| --- | --- |\n")

	count := 0
	for _, tag := range AllFeatures {
		if !keep[tag] {
			continue
		}
		info, ok := featureDescriptions[tag]
		if !ok {
			continue
		}
		sb.WriteString("| ")
		sb.WriteString(info.label)
		sb.WriteString(" | ")
		sb.WriteString(info.desc)
		sb.WriteString(" |\n")
		count++
	}

	if count == 0 {
		return "_Minimal configuration (no optional features)._"
	}
	return sb.String()
}

// buildFeatureSections generates per-feature documentation sections for the README.
func buildFeatureSections(features []string) string {
	if features == nil {
		return ""
	}

	expanded := ExpandFeatureDeps(features)
	keep := make(map[string]bool)
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}

	var sb strings.Builder

	if keep[FeatureAuth] {
		sb.WriteString(`### Authentication (Crooner)

This app uses [crooner](https://github.com/catgoose/crooner) for Azure AD / Entra ID authentication. Configure via environment variables:

- ` + "`AZURE_CLIENT_ID`" + `, ` + "`AZURE_CLIENT_SECRET`" + `, ` + "`AZURE_TENANT_ID`" + ` -- Azure app registration
- ` + "`AZURE_REDIRECT_URL`" + `, ` + "`AZURE_LOGIN_REDIRECT_URL`" + `, ` + "`AZURE_LOGOUT_REDIRECT_URL`" + ` -- OAuth flow URLs
- ` + "`SESSION_SECRET`" + ` -- session encryption key

Auth is disabled by default (` + "`CroonerDisabled = true`" + ` in config). Set it to ` + "`false`" + ` to enable.

`)
	}

	if keep[FeatureGraph] {
		sb.WriteString(`### Microsoft Graph API

Graph API integration is included for user data queries. Set the same Azure credentials as auth, plus:

- ` + "`AZURE_USER_REFRESH_HOUR`" + ` -- hour (0-23) for daily user cache sync
- ` + "`ENABLE_PHOTO_DOWNLOAD`" + ` -- download user photos from Graph

`)
	}

	if keep[FeatureSSE] {
		sb.WriteString(`### Server-Sent Events (SSE)

Real-time event broker with topic-based publish/subscribe. HTMX SSE extension is included for declarative event binding. Caddy is configured for SSE streaming support.

`)
	}

	if keep[FeatureCaddy] {
		sb.WriteString(`### Caddy (HTTPS)

Caddy provides TLS termination for local development. Certificates are generated during setup or can be provided manually. See the HTTPS Development Setup section for trust store installation.

`)
	}

	if keep[FeatureCapacitor] {
		sb.WriteString(`### Capacitor (Mobile)

Capacitor wraps the web app for native iOS/Android deployment. Configuration is in ` + "`capacitor.config.ts`" + `.

`)
	}

	if keep[FeatureOffline] || keep[FeatureSync] || keep[FeaturePWA] {
		sb.WriteString(`### Offline & Sync

`)
		if keep[FeaturePWA] {
			sb.WriteString("This app is configured as a **Progressive Web App** with offline support and data synchronization.\n\n")
		} else if keep[FeatureSync] {
			sb.WriteString("Data synchronization is enabled between client SQLite and server.\n\n")
		} else {
			sb.WriteString("Offline mode is enabled with service worker caching and a write queue.\n\n")
		}
	}

	return sb.String()
}

// buildTechStack generates a markdown table of technologies used by the app.
// Core entries are always included; conditional entries depend on selected features.
func buildTechStack(features []string) string {
	expanded := ExpandFeatureDeps(features)
	keep := make(map[string]bool)
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}

	var sb strings.Builder
	sb.WriteString("| Component | Purpose |\n")
	sb.WriteString("| --- | --- |\n")

	// Core (always included)
	sb.WriteString("| [Go](https://go.dev/) | Application server, single binary output |\n")
	sb.WriteString("| [Echo](https://echo.labstack.com/) | HTTP routing and middleware |\n")
	sb.WriteString("| [HTMX](https://htmx.org/) | Hypermedia interactions |\n")
	sb.WriteString("| [templ](https://templ.guide/) | Type-safe HTML templating |\n")
	sb.WriteString("| [Tailwind CSS](https://tailwindcss.com/) | Utility-first styling |\n")
	sb.WriteString("| [DaisyUI](https://daisyui.com/) | Semantic component classes with 30+ themes |\n")
	sb.WriteString("| [Hyperscript](https://hyperscript.org/) | Client-side DOM interactions |\n")
	sb.WriteString("| [SQLite](https://www.sqlite.org/) | Embedded database (dev), session storage |\n")

	// Conditional
	if keep[FeatureMSSQL] {
		sb.WriteString("| [SQL Server](https://www.microsoft.com/sql-server) | Production database |\n")
	}
	if keep[FeaturePostgres] {
		sb.WriteString("| [PostgreSQL](https://www.postgresql.org/) | Production database |\n")
	}
	if keep[FeatureCaddy] {
		sb.WriteString("| [Caddy](https://caddyserver.com/) | HTTPS reverse proxy |\n")
	}
	if keep[FeatureAlpine] {
		sb.WriteString("| [Alpine.js](https://alpinejs.dev/) | Client-side state management |\n")
	}

	// Always included (dev tools)
	sb.WriteString("| [Air](https://github.com/air-verse/air) | Live reload for development |\n")
	sb.WriteString("| [Mage](https://magefile.org/) | Build automation (Go-based) |\n")

	return sb.String()
}

// buildQuickStart generates the Quick Start section with binary name and port.
func buildQuickStart(binaryName, appTLSPort string) string {
	var sb strings.Builder

	sb.WriteString("### From Source\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("go build -o " + binaryName + " .\n")
	sb.WriteString("./" + binaryName + "\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### From Docker\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("docker build -t " + binaryName + " .\n")
	sb.WriteString("docker run -p " + appTLSPort + ":" + appTLSPort + " " + binaryName + "\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### From Release Binary\n\n")
	sb.WriteString("Download the latest release for your platform from the [Releases](../../releases) page:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Linux\n")
	sb.WriteString("chmod +x " + binaryName + "-linux-amd64\n")
	sb.WriteString("./" + binaryName + "-linux-amd64\n")
	sb.WriteString("\n")
	sb.WriteString("# Windows\n")
	sb.WriteString(binaryName + "-windows-amd64.exe\n")
	sb.WriteString("```\n\n")

	sb.WriteString("Override the default port:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("SERVER_LISTEN_PORT=8080 ./" + binaryName + "-linux-amd64\n")
	sb.WriteString("```")

	return sb.String()
}

// buildEnvTable generates a markdown table of environment variables based on features.
func buildEnvTable(features []string, appName, appTLSPort string) string {
	expanded := ExpandFeatureDeps(features)
	keep := make(map[string]bool)
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}

	var sb strings.Builder
	sb.WriteString("| Variable | Description | Default |\n")
	sb.WriteString("| --- | --- | --- |\n")

	// Always included
	sb.WriteString("| `SERVER_LISTEN_PORT` | Echo server port | " + appTLSPort + " |\n")
	sb.WriteString("| `APP_NAME` | Application name | " + appName + " |\n")
	sb.WriteString("| `LOG_LEVEL` | DEBUG, INFO, WARN, ERROR | INFO |\n")
	sb.WriteString("| `ENABLE_DATABASE` | Enable SQL backend | false |\n")
	sb.WriteString("| `DATABASE_URL` | Database connection string | sqlite:///db/app.db |\n")

	// Auth
	if keep[FeatureAuth] {
		sb.WriteString("| `SESSION_SECRET` | Session encryption key | (required with auth) |\n")
		sb.WriteString("| `OIDC_ISSUER_URL` | OIDC provider issuer URL | -- |\n")
		sb.WriteString("| `OIDC_CLIENT_ID` | OIDC client ID | -- |\n")
		sb.WriteString("| `OIDC_CLIENT_SECRET` | OIDC client secret | -- |\n")
	}

	// CSRF
	if keep[FeatureCSRF] {
		sb.WriteString("| `CSRF_ROTATE_PER_REQUEST` | Rotate CSRF token per request | false |\n")
	}

	// Graph
	if keep[FeatureGraph] {
		sb.WriteString("| `ENABLE_PHOTO_DOWNLOAD` | Download user photos from Graph | false |\n")
	}

	return sb.String()
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// composeSetupEnv processes "# setup:env " comment tags in an env file.
// Each tagged line is uncommented (prefix stripped) and the immediately
// following line (the literal default value) is removed.
func composeSetupEnv(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	skipNext := false
	for _, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(line, setupEnvPrefix) {
			out = append(out, strings.TrimPrefix(line, setupEnvPrefix))
			skipNext = true
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// replaceE2EWithSmoke removes all demo-specific e2e spec files and generates a
// minimal smoke test that verifies the home page loads and the health endpoint
// returns the configured app name (#356).
func replaceE2EWithSmoke(dir, appName string) {
	e2eDir := filepath.Join(dir, "e2e")
	entries, err := os.ReadDir(e2eDir)
	if err != nil {
		return // no e2e directory — nothing to do
	}

	// Remove all *.spec.ts files (demo-specific tests).
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".spec.ts") {
			_ = os.Remove(filepath.Join(e2eDir, e.Name()))
		}
	}

	binaryName := binaryNameFromApp(appName)

	// Generate a minimal smoke test.
	smoke := `import { test, expect } from "@playwright/test";
import { navigateTo } from "./helpers";

test.describe("` + appName + ` Smoke Tests", () => {
  test("home page loads", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page).toHaveTitle(/.+/);
  });

  test("health endpoint returns OK with correct app name", async ({ request }) => {
    const resp = await request.get("/health");
    expect(resp.ok()).toBe(true);
    const body = await resp.json();
    expect(body.status).toBe("healthy");
    expect(body.name).toBe("` + binaryName + `");
  });

  test("navbar is present", async ({ page }) => {
    await navigateTo(page, "/");
    await expect(page.locator("nav")).toBeVisible();
  });
});
`
	_ = os.WriteFile(filepath.Join(e2eDir, "smoke.spec.ts"), []byte(smoke), 0644)

	// Rewrite helpers.ts to remove demo-specific resetDB helper.
	helpers := `import { type Page, expect } from "@playwright/test";

/** Wait for HTMX to finish all pending requests. */
export async function waitForHtmx(page: Page) {
  await page.waitForFunction(
    () =>
      typeof (window as any).htmx !== "undefined" &&
      (document.querySelectorAll(".htmx-request").length === 0),
    { timeout: 10_000 },
  );
}

/** Navigate to a page and assert it loaded (no server error). */
export async function navigateTo(page: Page, path: string) {
  const resp = await page.goto(path);
  expect(resp?.ok(), ` + "`" + `Expected 2xx for ${path}, got ${resp?.status()}` + "`" + `).toBe(
    true,
  );
}
`
	_ = os.WriteFile(filepath.Join(e2eDir, "helpers.ts"), []byte(helpers), 0644)
}

// replaceDefaultFavicons copies the generic favicons from images/default/ over
// the demo-specific ones in images/, then removes the default/ directory.
func replaceDefaultFavicons(dir string) {
	imgDir := filepath.Join(dir, "web", "assets", "public", "images")
	defaultDir := filepath.Join(imgDir, "default")
	entries, err := os.ReadDir(defaultDir)
	if err != nil {
		return // default directory doesn't exist — nothing to do
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(defaultDir, e.Name())
		dst := filepath.Join(imgDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(dst, data, 0644)
	}
	// Remove demo-only assets that have no default replacement.
	for _, name := range []string{
		"android-chrome-192x192.png",
		"android-chrome-512x512.png",
		"site.webmanifest",
	} {
		if _, err := os.Stat(filepath.Join(defaultDir, name)); os.IsNotExist(err) {
			_ = os.Remove(filepath.Join(imgDir, name))
		}
	}
	_ = os.RemoveAll(defaultDir)
}

// isTextFile returns true for file extensions that may contain setup markers.
func isTextFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".templ", ".mod", ".sum", ".toml", ".yaml", ".yml", ".json", ".js", ".ts", ".css", ".html", ".md", ".txt", ".rb", ".webmanifest":
		return true
	}
	// Extensionless files that are known text.
	switch filepath.Base(path) {
	case "Appfile", "Gemfile", "Fastfile", "Matchfile", "Pluginfile":
		return true
	}
	return false
}

// removeEmptyDirs walks bottom-up and removes directories that became empty.
func removeEmptyDirs(dir string) error {
	var dirs []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != dir {
			name := info.Name()
			if name == ".git" || name == TemplateSetupDir {
				return filepath.SkipDir
			}
			dirs = append(dirs, path)
		}
		return nil
	})
	// Walk collected dirs in reverse (deepest first) for bottom-up removal
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}
	return nil
}

// CopyRepoTo copies directory tree from src to dest, skipping named dirs and symlinks.
func CopyRepoTo(src, dest string, excludeDirs []string) error {
	exclude := make(map[string]bool)
	for _, d := range excludeDirs {
		exclude[d] = true
	}
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	return filepath.Walk(srcAbs, func(path string, info os.FileInfo, errWalk error) error {
		if errWalk != nil {
			return errWalk
		}
		// Skip symlinks — they may point outside the repo.
		if linfo, err := os.Lstat(path); err == nil && linfo.Mode()&os.ModeSymlink != 0 {
			if linfo.IsDir() || info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dest, info.Mode())
		}
		if info.IsDir() && exclude[filepath.Base(path)] {
			return filepath.SkipDir
		}
		destPath := filepath.Join(dest, rel)
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
