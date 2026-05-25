//go:build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"catgoose/dothog/internal/setup"

	"github.com/charmbracelet/huh"
)

const templateModulePath = "catgoose/dothog"

var featureLabels = map[string]string{
	// Core
	setup.FeatureSessionSettings: "Session Settings (SQLite)",
	setup.FeatureCSRF:            "CSRF Protection",
	// Auth
	setup.FeatureAuth:   "Auth (Crooner)",
	setup.FeatureGraph:  "Graph API",
	setup.FeatureAvatar: "Avatar Photos (requires Graph)",
	// App data — the chuck-backed repository layer is hidden behind these
	// choices; selecting a production engine pulls the layer in via featureDeps.
	setup.FeatureMSSQL:    "MSSQL (Microsoft SQL Server)",
	setup.FeaturePostgres: "PostgreSQL",
	// Real-time — SSE pulls in the (hidden) Caddy HTTPS/H3 front-proxy via
	// featureDeps. There's no standalone Caddy selection any more.
	setup.FeatureSSE: "SSE (Server-Sent Events broker, includes Caddy HTTPS/H3 dev proxy)",
	// Mobile
	setup.FeatureCapacitor: "Capacitor (mobile wrapper)",
	// Demo
	setup.FeatureDemo: "Demo Content",
}

var featureLabelOrder = []string{
	// Core
	setup.FeatureSessionSettings,
	setup.FeatureCSRF,
	// Auth
	setup.FeatureAuth,
	setup.FeatureGraph,
	setup.FeatureAvatar,
	// App data
	setup.FeatureMSSQL,
	setup.FeaturePostgres,
	// Real-time
	setup.FeatureSSE,
	// Mobile
	setup.FeatureCapacitor,
	// Demo
	setup.FeatureDemo,
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Setup is the template repo's scaffold entrypoint for both the interactive
// wizard and the flag-driven setup flow.
// Example:
//
//	go tool mage setup
//	go tool mage setup -n "My App" -m "github.com/you/my-app" -p 12345
//	go tool mage setup -n "My App" --features auth,mssql
//	go tool mage setup -n "My App" --features none
func Setup() error {
	scriptArgs := setupScriptArgsFromCLI()
	parsed, hasFlags, helpPrinted, err := parseSetupFlags(scriptArgs)
	if err != nil {
		return err
	}
	if helpPrinted {
		// Mage parses every os.Args entry as a target name, so returning nil
		// here would let Mage fall through and report `--help` as an unknown
		// target (exit 2). Exit clean after the custom usage prints so
		// `go tool mage setup --help` behaves like every other CLI's help.
		os.Exit(0)
	}

	if hasFlags && parsed != nil {
		fmt.Println("Running template setup...")
		if err := maybeInstallCaddyForSetup(".", parsed); err != nil {
			return err
		}
		if err := initializeGitRepo("."); err != nil {
			return err
		}
		if err := setup.Run(context.Background(), ".", *parsed); err != nil {
			return err
		}
		if goModulePath() != templateModulePath {
			if err := cleanupTemplateFiles(); err != nil {
				fmt.Println("Warning: cleanup failed:", err)
			}
			fmt.Println("Template setup files removed.")
		}
		return nil
	}

	var absTarget string
	for {
		target, err := huhInput("Target directory", "e.g. ../my-app or /path/to/project", "")
		if err != nil {
			return err
		}
		target = strings.TrimSpace(target)
		if target == "" {
			return errors.New("target directory is required")
		}
		if strings.HasPrefix(target, "~") {
			home, _ := os.UserHomeDir()
			target = home + target[1:]
		}
		absTarget, err = filepath.Abs(target)
		if err != nil {
			return err
		}
		wd, _ := os.Getwd()
		if filepath.Clean(absTarget) == filepath.Clean(wd) {
			return errors.New("target directory cannot be the current directory")
		}
		info, err := os.Stat(absTarget)
		switch {
		case err == nil && !info.IsDir():
			return errors.New("target path exists and is not a directory")
		case err == nil:
			entries, err := os.ReadDir(absTarget)
			if err != nil {
				return err
			}
			if len(entries) > 0 {
				ok, err := huhConfirm("Target directory exists and is not empty. Remove directory and replace it?")
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				if err := os.RemoveAll(absTarget); err != nil {
					return fmt.Errorf("removing target directory: %w", err)
				}
			}
		case errors.Is(err, os.ErrNotExist):
		default:
			return err
		}
		break
	}
	opts, err := runWizard(absTarget)
	if err != nil {
		return err
	}
	if opts == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
		return err
	}
	if err := setup.CopyRepoTo(".", absTarget, []string{".git", ".claude", ".cursor", "bin", "build", "log", "node_modules", "test-results", "tmp", "localhost.crt", "localhost.key"}); err != nil {
		return fmt.Errorf("copying template: %w", err)
	}
	if err := initializeGitRepo(absTarget); err != nil {
		return err
	}
	// Remove setup-only files before running setup so that go mod tidy
	// does not see the rewritten mage_setup.go import.
	_ = os.RemoveAll(filepath.Join(absTarget, setup.TemplateSetupDir))
	_ = os.RemoveAll(filepath.Join(absTarget, "internal", "setup"))
	_ = os.Remove(filepath.Join(absTarget, "mage_setup.go"))
	if err := maybeInstallCaddyForSetup(absTarget, opts); err != nil {
		return err
	}
	if err := setup.Run(context.Background(), absTarget, *opts); err != nil {
		return err
	}
	fmt.Println("Setup complete in", absTarget)
	return nil
}

// presets describe user-facing preset bundles. The "database" tag is hidden
// and dependency-driven (MSSQL/PostgreSQL imply it via featureDeps), so it is
// never listed in a preset directly. Presets that don't select a production
// engine ship without the chuck-backed app-data layer and run on the
// framework-internal SQLite stores only.
var presets = map[string][]string{
	"internal":           {setup.FeatureAuth, setup.FeatureCSRF, setup.FeatureSessionSettings, setup.FeatureSSE},
	"microsoft-internal": {setup.FeatureSessionSettings, setup.FeatureCSRF, setup.FeatureAuth, setup.FeatureGraph, setup.FeatureAvatar, setup.FeatureMSSQL, setup.FeatureSSE},
	"public":             {setup.FeatureSessionSettings, setup.FeatureSSE},
	"demo":               setup.AllFeatures,
	"minimal":            {},
}

func runWizard(targetDir string) (*setup.Options, error) {
	var (
		appName    string
		modulePath string
		basePort   string
		features   []string
		force      bool
		confirm    = true
		preset     string
		customize  bool
		// Guided wizard answers
		dbEngine      string // "", "mssql", "postgres" — empty leaves derived app on framework-internal SQLite stores
		wantSessions  bool
		wantAuth      bool
		wantGraph     bool
		wantAvatar    bool
		wantSSE       bool
		wantCapacitor bool
		wantDemo      bool
	)

	currentModule := goModulePath()
	defaultPort := fmt.Sprintf("%d", randomBasePort())
	appNameSuggestion := appNameSuggestionFromTargetDir(targetDir)
	if appNameSuggestion != "" {
		appName = appNameSuggestion
	}
	needsForce := currentModule != "" && currentModule != templateModulePath

	// ── Step 1: App configuration ──────────────────────────────────

	appForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("App name").
				Placeholder("My App").
				Value(&appName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("app name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Module path").
				PlaceholderFunc(func() string {
					return modulePathSuggestion(appName, appNameSuggestion)
				}, &appName).
				Value(&modulePath),
			huh.NewInput().
				Title("Base port").
				Placeholder("5-digit port < 60000").
				Description(fmt.Sprintf("APP_HTTP=BASE, TEMPL_HTTP=BASE+1, CADDY_TLS=BASE+2 (default: %s)", defaultPort)).
				Value(&basePort),
		).Title("App Configuration"),
	)
	if err := appForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
		return nil, err
	}

	// ── Step 2: Preset or guided ───────────────────────────────────

	presetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What are you building?").
				Options(
					huh.NewOption("Internal tool — auth, sessions, SSE", "internal"),
					huh.NewOption("Microsoft Internal — auth, Graph, avatar, MSSQL app data, SSE", "microsoft-internal"),
					huh.NewOption("Public site — sessions, SSE", "public"),
					huh.NewOption("Demo/playground — everything enabled", "demo"),
					huh.NewOption("Minimal — bare HTMX app", "minimal"),
					huh.NewOption("Pick from list (flat checklist)", "flat"),
					huh.NewOption("Let me choose (guided wizard)", "guided"),
				).
				Value(&preset),
		).Title("Feature Preset"),
	)
	if err := presetForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
		return nil, err
	}

	if preset == "flat" {
		// ── Flat checklist: sensible defaults pre-selected ─────────
		flatDefaults := map[string]bool{
			setup.FeatureSessionSettings: true,
			setup.FeatureCSRF:            true,
			setup.FeatureSSE:             true,
		}
		var featureOptions []huh.Option[string]
		for _, tag := range featureLabelOrder {
			opt := huh.NewOption(featureLabels[tag], tag)
			if flatDefaults[tag] {
				opt = opt.Selected(true)
			}
			featureOptions = append(featureOptions, opt)
		}
		flatForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Features").
					Description("Dependencies will be auto-included after selection").
					Options(featureOptions...).
					Value(&features),
			).Title("Select Features"),
		)
		if err := flatForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println("Setup cancelled.")
				return nil, nil
			}
			return nil, err
		}
	} else if preset == "guided" {
		// ── Guided wizard: ask about dependencies first ────────────

		guidedForm := huh.NewForm(
			// MSSQL / PostgreSQL are the user-facing app-data choices.
			// Leaving this at "None" leaves the derived app on its
			// framework-internal SQLite stores; no separate app-data choice
			// is exposed.
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("MSSQL or PostgreSQL support?").
					Description("Pick one if you deploy to it. Leave at None otherwise.").
					Options(
						huh.NewOption("None", ""),
						huh.NewOption("MSSQL", "mssql"),
						huh.NewOption("PostgreSQL", "postgres"),
					).
					Value(&dbEngine),
			).Title("App Data"),

			// Sessions
			huh.NewGroup(
				huh.NewConfirm().Title("Need user sessions? (theme persistence, settings)").Value(&wantSessions),
			).Title("Sessions"),

			// Auth
			huh.NewGroup(
				huh.NewConfirm().Title("Need authentication? (Crooner)\n  CSRF protection will be auto-included").Value(&wantAuth),
			).Title("Authentication"),

			huh.NewGroup(
				huh.NewConfirm().Title("Need Microsoft Graph API?").Value(&wantGraph),
			).Title("Graph API").WithHideFunc(func() bool { return !wantAuth }),

			huh.NewGroup(
				huh.NewConfirm().Title("Need user photos from Graph?").Value(&wantAvatar),
			).Title("Avatar Photos").WithHideFunc(func() bool { return !wantGraph }),

			// Real-time
			huh.NewGroup(
				huh.NewConfirm().Title("Need real-time updates (SSE)?\n  Caddy HTTPS/H3 dev proxy will be auto-included.").Value(&wantSSE),
			).Title("Real-time"),

			// Mobile
			huh.NewGroup(
				huh.NewConfirm().Title("Need mobile app wrapper (Capacitor)?").Value(&wantCapacitor),
			).Title("Mobile"),

			// Demo
			huh.NewGroup(
				huh.NewConfirm().Title("Include demo content?").Value(&wantDemo),
			).Title("Demo"),
		)
		if err := guidedForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println("Setup cancelled.")
				return nil, nil
			}
			return nil, err
		}

		// Build features from guided answers. "database" is hidden and
		// dependency-driven, so only the production-engine tag needs to be
		// appended; ExpandFeatureDeps closes the implication. Empty dbEngine
		// leaves the app on framework-internal SQLite stores only.
		switch dbEngine {
		case "mssql":
			features = append(features, setup.FeatureMSSQL)
		case "postgres":
			features = append(features, setup.FeaturePostgres)
		}
		if wantSessions {
			features = append(features, setup.FeatureSessionSettings)
		}
		if wantAuth {
			features = append(features, setup.FeatureAuth, setup.FeatureCSRF)
		}
		if wantGraph {
			features = append(features, setup.FeatureGraph)
		}
		if wantAvatar {
			features = append(features, setup.FeatureAvatar)
		}
		if wantSSE {
			features = append(features, setup.FeatureSSE)
		}
		if wantCapacitor {
			features = append(features, setup.FeatureCapacitor)
		}
		if wantDemo {
			features = append(features, setup.FeatureDemo)
		}

	} else {
		// ── Preset selected: offer to customize ────────────────────

		features = make([]string, len(presets[preset]))
		copy(features, presets[preset])

		customizeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					TitleFunc(func() string {
						return fmt.Sprintf("%q preset includes: %s\n\nCustomize these selections?",
							preset, describeFeatures(features))
					}, &preset).
					Value(&customize),
			).Title("Review Preset"),
		)
		if err := customizeForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println("Setup cancelled.")
				return nil, nil
			}
			return nil, err
		}

		if customize {
			// Show flat checklist with preset pre-checked
			preSelected := make(map[string]bool, len(features))
			for _, f := range features {
				preSelected[f] = true
			}
			var featureOptions []huh.Option[string]
			for _, tag := range featureLabelOrder {
				label := featureLabels[tag]
				opt := huh.NewOption(label, tag)
				if preSelected[tag] {
					opt = opt.Selected(true)
				}
				featureOptions = append(featureOptions, opt)
			}

			features = nil // reset — multiselect will populate
			flatForm := huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Features").
						Description("Dependencies will be auto-included after selection").
						Options(featureOptions...).
						Value(&features),
				).Title("Customize Features"),
			)
			if err := flatForm.Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					fmt.Println("Setup cancelled.")
					return nil, nil
				}
				return nil, err
			}
		}
	}

	// ── Force confirm (if module already customized) ───────────────

	if needsForce {
		forceForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Module already customized (go.mod: %s). Run setup again with --force?", currentModule)).
					Value(&force),
			),
		)
		if err := forceForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println("Setup cancelled.")
				return nil, nil
			}
			return nil, err
		}
		if !force {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
	}

	// ── Final confirmation ─────────────────────────────────────────

	// Enforce feature dependencies
	// Feature implications (for example Avatar→Graph, MSSQL/Postgres→Database,
	// SSE→Caddy, Demo→Sessions) are closed by setup.ExpandFeatureDeps later in
	// setup.Run.

	appName = strings.TrimSpace(appName)
	resolvedModule := resolveModulePath(appName, modulePath, currentModule)
	resolvedPort := basePort
	if resolvedPort == "" {
		resolvedPort = defaultPort
	}

	if len(resolvedPort) != 5 {
		return nil, fmt.Errorf("BASE_PORT must be a 5-digit number, got: %s", resolvedPort)
	}
	var basePortNum int
	if _, err := fmt.Sscanf(resolvedPort, "%d", &basePortNum); err != nil || basePortNum >= 60000 {
		return nil, fmt.Errorf("BASE_PORT must be < 60000, got: %s", resolvedPort)
	}

	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Proceed with app %q, module %s, port %s, features: %s?",
					appName, resolvedModule, resolvedPort, describeFeatures(features))).
				Value(&confirm),
		),
	)
	if err := confirmForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
		return nil, err
	}
	if !confirm {
		fmt.Println("Setup cancelled.")
		return nil, nil
	}

	return &setup.Options{
		AppName:     appName,
		ModulePath:  resolvedModule,
		BasePort:    resolvedPort,
		Force:       force,
		Features:    features,
		ConfirmFunc: huhConfirm,
	}, nil
}

func resolveModulePath(appName, modulePathInput, _ string) string {
	modulePathInput = strings.TrimSpace(modulePathInput)
	if modulePathInput != "" {
		return modulePathInput
	}
	name := strings.TrimSpace(appName)
	if name == "" {
		return ""
	}
	return fmt.Sprintf("github.com/you/%s", binaryNameFromApp(name))
}

func modulePathSuggestion(appName, appNameSuggestion string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = strings.TrimSpace(appNameSuggestion)
	}
	if name == "" {
		return "github.com/you/my-app"
	}
	return fmt.Sprintf("github.com/you/%s", binaryNameFromApp(name))
}

func appNameSuggestionFromTargetDir(targetDir string) string {
	targetDir = strings.TrimSpace(targetDir)
	if targetDir == "" {
		return ""
	}
	cleaned := filepath.Clean(targetDir)
	base := filepath.Base(cleaned)
	switch base {
	case "", ".", string(filepath.Separator):
		return ""
	}
	parts := strings.FieldsFunc(base, func(r rune) bool {
		switch r {
		case '-', '_', ' ':
			return true
		default:
			return false
		}
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func initializeGitRepo(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return fmt.Errorf("git init in %s: %w", dir, err)
		}
		return fmt.Errorf("git init in %s: %w: %s", dir, err, msg)
	}
	return nil
}

func describeFeatures(features []string) string {
	if len(features) == 0 {
		return "none (bare HTMX app)"
	}
	if len(features) >= len(setup.AllFeatures) {
		return "all"
	}
	return strings.Join(features, ", ")
}

func hasFeature(features []string, tag string) bool {
	for _, f := range features {
		if f == tag {
			return true
		}
	}
	return false
}

func setupUsesCaddy(opts *setup.Options) bool {
	if opts == nil {
		return false
	}
	if opts.Features == nil {
		// nil Features means "the wizard hasn't built a list yet" — treat as
		// the full feature set, so Caddy installs as long as SSE pulls it in.
		return hasFeature(setup.ExpandFeatureDeps(setup.AllFeatures), setup.FeatureCaddy)
	}
	return hasFeature(setup.ExpandFeatureDeps(opts.Features), setup.FeatureCaddy)
}

func maybeInstallCaddyForSetup(projectDir string, opts *setup.Options) error {
	if !setupUsesCaddy(opts) {
		return nil
	}
	if _, err := os.Stat(caddyLocalBinIn(projectDir)); err == nil {
		fmt.Println("Using existing ./bin/caddy for this app.")
		return nil
	}
	fmt.Println("Installing Caddy into ./bin...")
	return installRepoLocalCaddy(projectDir)
}

func parseSetupFlags(args []string) (opts *setup.Options, hasFlags bool, helpPrinted bool, err error) {
	for _, a := range args {
		switch a {
		case "-n", "-m", "-p", "--force", "--features", "--platform", "-h", "--help":
			hasFlags = true
			break
		}
	}
	if !hasFlags {
		return nil, false, false, nil
	}
	for i, a := range args {
		if a == "-h" || a == "--help" {
			printSetupUsage()
			return nil, true, true, nil
		}
		if a == "-n" && i+1 < len(args) {
			opts = &setup.Options{AppName: args[i+1]}
			break
		}
	}
	if opts == nil {
		return nil, true, false, errors.New("APP_NAME is required; use -n APP_NAME")
	}
	for i, a := range args {
		if a == "-m" && i+1 < len(args) {
			opts.ModulePath = args[i+1]
			break
		}
	}
	for i, a := range args {
		if a == "-p" && i+1 < len(args) {
			opts.BasePort = args[i+1]
			break
		}
	}
	if opts.BasePort == "" {
		opts.BasePort = fmt.Sprintf("%d", randomBasePort())
	}
	for _, a := range args {
		if a == "--force" {
			opts.Force = true
			break
		}
	}
	// --features flag
	for i, a := range args {
		if a == "--features" && i+1 < len(args) {
			opts.Features = parseFeatureFlag(args[i+1])
			break
		}
	}
	// --platform flag (linux|windows; empty = autodetect)
	for i, a := range args {
		if a == "--platform" && i+1 < len(args) {
			opts.Platform = args[i+1]
			break
		}
	}
	opts.ConfirmFunc = huhConfirm
	return opts, true, false, nil
}

// parseFeatureFlag is defined in magefile.go (survives mage_setup.go removal).

func printSetupUsage() {
	fmt.Println(`Usage: go tool mage setup [-n APP_NAME] [-m MODULE_PATH] [-p BASE_PORT] [--features FEATURES] [--platform OS] [--force]

  -n APP_NAME        Human-readable app name (e.g. "My App"). Required.
  -m MODULE_PATH     Go module path (e.g. "github.com/you/my-app").
  -p BASE_PORT       5-digit base port < 60000; APP_HTTP_PORT=BASE_PORT, TEMPL_HTTP_PORT=BASE_PORT+1, CADDY_TLS_PORT=BASE_PORT+2.
  --features LIST    Comma-separated user-selectable tags to keep: auth,graph,avatar,mssql,postgres,sse,demo,session_settings,csrf,capacitor.
                     "all" = keep everything (default), "none" = bare HTMX app. Hidden tags resolve via featureDeps: MSSQL/PostgreSQL imply database; SSE implies the Caddy HTTPS/H3 dev proxy.
  --platform OS      Target host OS for the derived app's dev tooling: linux or windows. Defaults to the current host's GOOS.
  --force            Allow re-running setup even if module is already customized.`)
}

func setupScriptArgsFromCLI() []string {
	args := os.Args[1:]
	idx := -1
	for i, a := range args {
		if a == "setup" {
			idx = i
			break
		}
	}
	if idx == -1 || idx+1 >= len(args) {
		return nil
	}
	return args[idx+1:]
}

// huhConfirm is huhConfirmDefault with a default of false.
func huhConfirm(message string) (bool, error) {
	return huhConfirmDefault(message, false)
}

// huhConfirmDefault returns false (and a nil error) on user abort, matching how callers handle cancellation.
func huhConfirmDefault(message string, def bool) (bool, error) {
	confirmed := def
	err := huh.NewConfirm().
		Title(message).
		Affirmative("Yes").
		Negative("No").
		Value(&confirmed).
		Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	return confirmed, nil
}

// huhInput returns "" (and a nil error) on user abort.
func huhInput(title, placeholder, value string) (string, error) {
	result := value
	field := huh.NewInput().
		Title(title).
		Placeholder(placeholder).
		Value(&result)
	err := huh.NewForm(huh.NewGroup(field)).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func binaryNameFromApp(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, " ", "-")
}

func goModulePath() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}
	return ""
}

func randomBasePort() int {
	return 10000 + rand.Intn(60000-10000)
}

func cleanupTemplateFiles() error {
	if err := os.RemoveAll("_template_setup"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove _template_setup: %w", err)
	}
	if err := os.RemoveAll(filepath.Join("internal", "setup")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove internal/setup: %w", err)
	}
	if err := os.Remove("mage_setup.go"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove mage_setup.go: %w", err)
	}
	return nil
}
