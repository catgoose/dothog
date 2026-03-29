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
	"github.com/magefile/mage/sh"
)

const templateModulePath = "catgoose/dothog"

// featureLabels maps feature tags to human-readable labels.
var featureLabels = map[string]string{
	// Core
	setup.FeatureSessionSettings: "Session Settings (SQLite)",
	setup.FeatureCSRF:            "CSRF Protection",
	// Auth
	setup.FeatureAuth:            "Auth (Crooner)",
	setup.FeatureGraph:           "Graph API",
	setup.FeatureAvatar:          "Avatar Photos (requires Graph)",
	// Data
	setup.FeatureDatabase:        "Database (fraggle repository layer)",
	setup.FeatureMSSQL:           "MSSQL dialect",
	setup.FeaturePostgres:        "PostgreSQL dialect",
	// Real-time
	setup.FeatureSSE:             "SSE (requires Caddy)",
	setup.FeatureCaddy:           "Caddy (HTTPS)",
	// Navigation
	setup.FeatureLinkRelations:   "Link Relations (context bars, breadcrumbs, site map)",
	// Performance & Security
	setup.FeatureWebStandards:    "Web Standards (Server-Timing, Vary, Permissions-Policy, Early Hints)",
	setup.FeatureBrowserAPIs:     "Browser APIs (sendBeacon, BroadcastChannel)",
	// Mobile & Offline
	setup.FeatureCapacitor:       "Capacitor (mobile wrapper)",
	setup.FeatureOffline:         "Offline Mode (service worker, write queue)",
	setup.FeatureSync:            "Sync (offline data synchronization)",
	setup.FeaturePWA:             "PWA (Progressive Web App — offline + sync + mobile)",
	// Demo
	setup.FeatureDemo:            "Demo Content",
}

// featureLabelOrder is the display order for the feature multi-select.
var featureLabelOrder = []string{
	// Core
	setup.FeatureSessionSettings,
	setup.FeatureCSRF,
	// Auth
	setup.FeatureAuth,
	setup.FeatureGraph,
	setup.FeatureAvatar,
	// Data
	setup.FeatureDatabase,
	setup.FeatureMSSQL,
	setup.FeaturePostgres,
	// Real-time
	setup.FeatureSSE,
	setup.FeatureCaddy,
	// Navigation
	setup.FeatureLinkRelations,
	// Performance & Security
	setup.FeatureWebStandards,
	setup.FeatureBrowserAPIs,
	// Mobile & Offline
	setup.FeatureCapacitor,
	setup.FeatureOffline,
	setup.FeatureSync,
	setup.FeaturePWA,
	// Demo
	setup.FeatureDemo,
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Setup runs the template setup to initialize a new app.
// Example:
//
//	go tool mage setup
//	go tool mage setup -n "My App" -m "github.com/you/my-app" -p 12345
//	go tool mage setup -n "My App" --features auth,database
//	go tool mage setup -n "My App" --features none
func Setup() error {
	scriptArgs := setupScriptArgsFromCLI()
	parsed, hasFlags, helpPrinted, err := parseSetupFlags(scriptArgs)
	if err != nil {
		return err
	}
	if helpPrinted {
		return nil
	}

	if hasFlags && parsed != nil {
		fmt.Println("Running template setup...")
		if err := setup.Run(context.Background(), ".", *parsed); err != nil {
			return err
		}
		if goModulePath() == templateModulePath {
			return nil
		}
		cleanup, err := huhConfirm("Cleanup template files and setup helpers from this repo?")
		if err != nil {
			return err
		}
		if cleanup {
			if err := cleanupTemplateFiles(); err != nil {
				return err
			}
			fmt.Println("Template setup files removed.")
		}
		return nil
	}

	copyFirst, err := huhConfirmDefault("Copy template to a new directory before setting up?", true)
	if err != nil {
		return err
	}
	if copyFirst {
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
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		wd, _ := os.Getwd()
		if filepath.Clean(absTarget) == filepath.Clean(wd) {
			return errors.New("target directory cannot be the current directory")
		}
		if info, err := os.Stat(absTarget); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(absTarget)
			if len(entries) > 0 {
				ok, _ := huhConfirm("Target directory exists and is not empty. Overwrite?")
				if !ok {
					fmt.Println("Setup cancelled.")
					return nil
				}
			}
		}
		opts, err := runWizard()
		if err != nil {
			return err
		}
		if opts == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
			return err
		}
		// Ensure node_modules are present before copying
		if _, err := os.Stat("package-lock.json"); err == nil {
			if err := sh.Run("npm", "ci"); err != nil {
				return fmt.Errorf("npm ci: %w", err)
			}
		}
		if err := setup.CopyRepoTo(".", absTarget, []string{".git", "bin", "build", "tmp"}); err != nil {
			return fmt.Errorf("copying template: %w", err)
		}
		gitInit, _ := huhConfirm("Run git init in the new directory?")
		if gitInit {
			cmd := exec.Command("git", "init")
			cmd.Dir = absTarget
			_ = cmd.Run()
		}
		// Remove setup-only files before running setup so that go mod tidy
		// does not see the rewritten mage_setup.go import.
		_ = os.RemoveAll(filepath.Join(absTarget, setup.TemplateSetupDir))
		_ = os.RemoveAll(filepath.Join(absTarget, "internal", "setup"))
		_ = os.Remove(filepath.Join(absTarget, "mage_setup.go"))
		if err := setup.Run(context.Background(), absTarget, *opts); err != nil {
			return err
		}
		fmt.Println("Setup complete in", absTarget)
		return nil
	}

	opts, err := runWizard()
	if err != nil {
		return err
	}
	if opts == nil {
		return nil
	}
	fmt.Println("Running template setup...")
	if err := setup.Run(context.Background(), ".", *opts); err != nil {
		return err
	}
	if goModulePath() == templateModulePath {
		return nil
	}
	cleanup, err := huhConfirm("Cleanup template files and setup helpers from this repo?")
	if err != nil {
		return err
	}
	if cleanup {
		if err := cleanupTemplateFiles(); err != nil {
			return err
		}
		fmt.Println("Template setup files removed. You can delete mage_setup.go if you no longer need the setup target.")
	}
	return nil
}

// presets maps preset names to their default feature sets.
var presets = map[string][]string{
	"internal": {setup.FeatureAuth, setup.FeatureCSRF, setup.FeatureDatabase, setup.FeatureSessionSettings, setup.FeatureSSE, setup.FeatureCaddy, setup.FeatureLinkRelations, setup.FeatureWebStandards},
	"public":   {setup.FeatureSessionSettings, setup.FeatureSSE, setup.FeatureCaddy, setup.FeatureLinkRelations, setup.FeatureWebStandards, setup.FeatureBrowserAPIs},
	"demo":     setup.AllFeatures,
	"minimal":  {},
}

func runWizard() (*setup.Options, error) {
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
		dbDialect    string // "sqlite", "mssql", "postgres", "sqlite+mssql", "sqlite+postgres"
		wantSessions bool
		wantAuth     bool
		wantGraph    bool
		wantAvatar   bool
		wantSSE      bool
		wantLinks    bool
		wantStandards bool
		wantAPIs     bool
		wantCapacitor bool
		wantOffline  bool
		wantSync     bool
		wantPWA      bool
		wantDemo     bool
	)

	currentModule := goModulePath()
	defaultPort := fmt.Sprintf("%d", randomBasePort())
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
					name := strings.TrimSpace(appName)
					if name == "" {
						return "github.com/you/my-app"
					}
					return fmt.Sprintf("github.com/you/%s", binaryNameFromApp(name))
				}, &appName).
				Value(&modulePath),
			huh.NewInput().
				Title("Base port").
				Placeholder("5-digit port < 60000").
				Description(fmt.Sprintf("APP_TLS_PORT=BASE, TEMPL_HTTP=BASE+1, CADDY_TLS=BASE+2 (default: %s)", defaultPort)).
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
					huh.NewOption("Internal tool — auth, database, sessions, SSE, link relations, web standards", "internal"),
					huh.NewOption("Public site — sessions, link relations, web standards, browser APIs", "public"),
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
		// ── Flat checklist: nothing pre-selected ───────────────────
		var featureOptions []huh.Option[string]
		for _, tag := range featureLabelOrder {
			featureOptions = append(featureOptions, huh.NewOption(featureLabels[tag], tag))
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
			// Database
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Database dialect").
					Description("SQLite is always included for dev/demo. Pick a production dialect if needed.").
					Options(
						huh.NewOption("SQLite only (dev/demo)", "sqlite"),
						huh.NewOption("SQLite + MSSQL (dev locally, deploy to SQL Server)", "sqlite+mssql"),
						huh.NewOption("SQLite + PostgreSQL (dev locally, deploy to Postgres)", "sqlite+postgres"),
						huh.NewOption("MSSQL only", "mssql"),
						huh.NewOption("PostgreSQL only", "postgres"),
					).
					Value(&dbDialect),
			).Title("Database"),

			// Navigation — ask first, auto-includes sessions
			huh.NewGroup(
				huh.NewConfirm().Title("Want link relations? (context bars, breadcrumbs, site map)\n  Session settings will be auto-included").Value(&wantLinks),
			).Title("Navigation"),

			// Sessions — only ask if link relations didn't already include it
			huh.NewGroup(
				huh.NewConfirm().Title("Need user sessions? (theme persistence, settings)").Value(&wantSessions),
			).Title("Sessions").WithHideFunc(func() bool { return wantLinks }),

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
				huh.NewConfirm().Title("Need real-time updates (SSE)?\n  Caddy HTTPS will be auto-included for HTTP/2").Value(&wantSSE),
			).Title("Real-time"),

			// Performance & Security
			huh.NewGroup(
				huh.NewConfirm().Title("Want web standards headers?\n  Server-Timing, Vary, Permissions-Policy, 103 Early Hints").Value(&wantStandards),
				huh.NewConfirm().Title("Want browser APIs?\n  sendBeacon analytics, BroadcastChannel cross-tab sync\n  SSE will be auto-included").Value(&wantAPIs),
			).Title("Performance & Security"),

			// Mobile & Offline
			huh.NewGroup(
				huh.NewConfirm().Title("Need mobile app wrapper (Capacitor)?").Value(&wantCapacitor),
				huh.NewConfirm().Title("Need offline support? (service worker, write queue)\n  Capacitor will be auto-included").Value(&wantOffline),
				huh.NewConfirm().Title("Need data sync?\n  Offline will be auto-included").Value(&wantSync),
				huh.NewConfirm().Title("Want full PWA?\n  Includes offline + sync + capacitor").Value(&wantPWA),
			).Title("Mobile & Offline"),

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

		// Build features from guided answers
		switch dbDialect {
		case "mssql":
			features = append(features, setup.FeatureDatabase, setup.FeatureMSSQL)
		case "postgres":
			features = append(features, setup.FeatureDatabase, setup.FeaturePostgres)
		case "sqlite+mssql":
			features = append(features, setup.FeatureDatabase, setup.FeatureMSSQL)
		case "sqlite+postgres":
			features = append(features, setup.FeatureDatabase, setup.FeaturePostgres)
		// "sqlite" — database is implicit, nothing extra needed
		}
		if wantSessions { features = append(features, setup.FeatureSessionSettings) }
		if wantAuth { features = append(features, setup.FeatureAuth, setup.FeatureCSRF) }
		if wantGraph { features = append(features, setup.FeatureGraph) }
		if wantAvatar { features = append(features, setup.FeatureAvatar) }
		if wantSSE { features = append(features, setup.FeatureSSE, setup.FeatureCaddy) }
		if wantLinks { features = append(features, setup.FeatureLinkRelations, setup.FeatureSessionSettings) }
		if wantStandards { features = append(features, setup.FeatureWebStandards) }
		if wantAPIs { features = append(features, setup.FeatureBrowserAPIs, setup.FeatureSSE, setup.FeatureCaddy) }
		if wantCapacitor { features = append(features, setup.FeatureCapacitor) }
		if wantOffline { features = append(features, setup.FeatureOffline, setup.FeatureCapacitor) }
		if wantSync { features = append(features, setup.FeatureSync, setup.FeatureOffline, setup.FeatureCapacitor) }
		if wantPWA { features = append(features, setup.FeaturePWA, setup.FeatureSync, setup.FeatureOffline, setup.FeatureCapacitor) }
		if wantDemo { features = append(features, setup.FeatureDemo) }

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
	features = enforceFeatureDeps(features)

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

// resolveModulePath determines the final module path from user input and defaults.
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

// describeFeatures returns a human-readable summary of selected features.
func describeFeatures(features []string) string {
	if len(features) == 0 {
		return "none (bare HTMX app)"
	}
	if len(features) >= len(setup.AllFeatures) {
		return "all"
	}
	return strings.Join(features, ", ")
}

// enforceFeatureDeps auto-includes required dependencies.
func enforceFeatureDeps(features []string) []string {
	tagSet := make(map[string]bool)
	for _, f := range features {
		tagSet[f] = true
	}
	if tagSet[setup.FeatureSSE] && !tagSet[setup.FeatureCaddy] {
		features = append(features, setup.FeatureCaddy)
		fmt.Println("SSE requires Caddy for proxying; Caddy auto-included.")
	}
	if tagSet[setup.FeatureAvatar] && !tagSet[setup.FeatureGraph] {
		features = append(features, setup.FeatureGraph)
		fmt.Println("Avatar requires Graph API; Graph auto-included.")
	}
	return features
}

func hasFeature(features []string, tag string) bool {
	for _, f := range features {
		if f == tag {
			return true
		}
	}
	return false
}

func parseSetupFlags(args []string) (opts *setup.Options, hasFlags bool, helpPrinted bool, err error) {
	for _, a := range args {
		switch a {
		case "-n", "-m", "-p", "--force", "--no-caddy", "--features", "-h", "--help":
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
	for _, a := range args {
		if a == "--no-caddy" {
			opts.NoCaddy = true
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
	// --no-caddy is a deprecated alias: if features not explicitly set,
	// apply it by setting features to all-except-caddy
	if opts.NoCaddy && opts.Features == nil {
		opts.Features = make([]string, 0, len(setup.AllFeatures)-1)
		for _, f := range setup.AllFeatures {
			if f != setup.FeatureCaddy {
				opts.Features = append(opts.Features, f)
			}
		}
	}
	opts.ConfirmFunc = huhConfirm
	return opts, true, false, nil
}

// parseFeatureFlag is defined in magefile.go (survives mage_setup.go removal).

func printSetupUsage() {
	fmt.Println(`Usage: go tool mage setup [-n APP_NAME] [-m MODULE_PATH] [-p BASE_PORT] [--features FEATURES] [--no-caddy] [--force]

  -n APP_NAME        Human-readable app name (e.g. "My App"). Required.
  -m MODULE_PATH     Go module path (e.g. "github.com/you/my-app").
  -p BASE_PORT       5-digit base port < 60000; APP_TLS_PORT=BASE_PORT, TEMPL_HTTP_PORT=BASE_PORT+1, CADDY_TLS_PORT=BASE_PORT+2.
  --features LIST    Comma-separated feature tags to keep: auth,graph,avatar,database,sse,caddy,demo,alpine.
                     "all" = keep everything (default), "none" = bare HTMX app.
  --no-caddy         Deprecated. Equivalent to omitting caddy from --features.
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

// huhConfirm prompts the user with a yes/no confirmation using huh.
func huhConfirm(message string) (bool, error) {
	return huhConfirmDefault(message, false)
}

// huhConfirmDefault prompts the user with a yes/no confirmation using huh,
// with a configurable default value.
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

// huhInput prompts the user for text input using huh.
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
