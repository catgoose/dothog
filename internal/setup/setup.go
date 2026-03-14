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
	FeatureSSE      = "sse"
	FeatureCaddy    = "caddy"
	FeatureAvatar   = "avatar"
	FeatureDemo             = "demo"
	FeatureSessionSettings  = "session_settings"
)

// AllFeatures lists every selectable feature tag.
// "database" is always included (implied by the base template) and is not user-selectable.
var AllFeatures = []string{FeatureAuth, FeatureGraph, FeatureDatabase, FeatureMSSQL, FeatureSSE, FeatureCaddy, FeatureAvatar, FeatureDemo, FeatureSessionSettings}

// ImplicitFeatures are always selected and not presented to the user.
// "database" is implicit because SQLite is the base database engine.
var ImplicitFeatures = []string{FeatureDatabase}

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

	binaryNameRe := regexp.MustCompile(`binaryName\s*=\s*"dothog"`)
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

	templateReadme := filepath.Join(dir, TemplateSetupDir, "README.template.md")
	if data, err := os.ReadFile(templateReadme); err == nil {
		content := string(data)
		content = strings.ReplaceAll(content, "{{APP_TLS_PORT}}", appTLSPort)
		content = strings.ReplaceAll(content, "{{TEMPL_HTTP_PORT}}", templHTTPPort)
		content = strings.ReplaceAll(content, "{{CADDY_TLS_PORT}}", caddyTLSPort)
		content = strings.ReplaceAll(content, "{{APP_NAME}}", opts.AppName)
		content = strings.ReplaceAll(content, "{{MODULE_PATH}}", modulePath)
		content = strings.ReplaceAll(content, "{{TEMPLATE_REF}}", "the template")
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0644); err != nil {
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
		for _, envTarget := range []string{".env.dev", ".env.development"} {
			if err := os.WriteFile(filepath.Join(dir, envTarget), []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(dir, "package-lock.json")); err == nil {
		content := strings.ReplaceAll(string(data), `"name": "dothog"`, `"name": "`+binaryName+`"`)
		content = regexp.MustCompile(`"name":\s*"dothog"`).ReplaceAllString(content, `"name": "`+binaryName+`"`)
		if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(content), 0644); err != nil {
			return err
		}
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content := regexp.MustCompile(`(?m)^dothog$`).ReplaceAllString(string(data), binaryName)
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	loggerPath := filepath.Join(dir, "internal", "logger", "logger.go")
	if data, err := os.ReadFile(loggerPath); err == nil {
		content := strings.ReplaceAll(string(data), `appLogFile = "dothog.log"`, `appLogFile = "`+binaryName+`.log"`)
		if err := os.WriteFile(loggerPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	_ = os.Remove(filepath.Join(dir, ".golangci.yml"))

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
		keep := make(map[string]bool)
		for _, f := range opts.Features {
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

	// Hardcoded binary/non-text files
	_ = os.Remove(filepath.Join(dir, "db", "demo.db"))
	if removeTags[FeatureSSE] {
		_ = os.Remove(filepath.Join(dir, "web", "assets", "public", "js", "htmx.ext.sse.js"))
	}
	if removeTags[FeatureCaddy] {
		_ = os.Remove(filepath.Join(dir, "config", "Caddyfile"))
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
		// Check internal packages: does the directory still exist?
		if strings.HasPrefix(imp.importPath, modulePath+"/") {
			relPkg := strings.TrimPrefix(imp.importPath, modulePath+"/")
			pkgDir := filepath.Join(baseDir, filepath.FromSlash(relPkg))
			if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
				removeLines[imp.lineIdx] = true
				continue
			}
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

// isTextFile returns true for file extensions that may contain setup markers.
func isTextFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".templ", ".mod", ".sum", ".toml", ".yaml", ".yml", ".json", ".js", ".ts", ".css", ".html", ".md", ".txt":
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

// CopyRepoTo copies directory tree from src to dest, skipping named dirs.
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
