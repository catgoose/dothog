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
	setup.FeatureAuth:            "Auth (Crooner)",
	setup.FeatureGraph:           "Graph API",
	setup.FeatureDatabase:        "Database (MSSQL)",
	setup.FeatureSSE:             "SSE (requires Caddy)",
	setup.FeatureCaddy:           "Caddy (HTTPS)",
	setup.FeatureAvatar:          "Avatar Photos (requires Graph)",
	setup.FeatureDemo:            "Demo Content",
	setup.FeatureSessionSettings: "Session Settings (SQLite)",
}

// featureLabelOrder is the display order for the feature multi-select.
var featureLabelOrder = []string{
	setup.FeatureAuth,
	setup.FeatureGraph,
	setup.FeatureAvatar,
	setup.FeatureDatabase,
	setup.FeatureSSE,
	setup.FeatureCaddy,
	setup.FeatureDemo,
	setup.FeatureSessionSettings,
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

func runWizard() (*setup.Options, error) {
	var (
		appName    string
		modulePath string
		basePort   string
		features   []string
		force      bool
		confirm    = true
	)

	currentModule := goModulePath()
	defaultPort := fmt.Sprintf("%d", randomBasePort())

	// Build feature options with preselection
	var featureOptions []huh.Option[string]
	for _, tag := range featureLabelOrder {
		label := featureLabels[tag]
		opt := huh.NewOption(label, tag)
		// Demo and Alpine are opt-in: not preselected by default
		if tag != setup.FeatureDemo && tag != setup.FeatureAlpine {
			opt = opt.Selected(true)
		}
		featureOptions = append(featureOptions, opt)
	}

	needsForce := currentModule != "" && currentModule != templateModulePath

	form := huh.NewForm(
		// Group 1: Basic app info
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

		// Group 2: Feature selection with dynamic descriptions
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Features").
				Description("SSE requires Caddy; Avatar requires Graph (auto-included if missing)").
				Options(featureOptions...).
				Value(&features),
		).Title("Feature Selection"),

		// Group 3: Force confirm — only shown when module is already customized
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Module already customized (go.mod: %s). Run setup again with --force?", currentModule)
				}, &appName).
				Value(&force),
		).WithHideFunc(func() bool {
			return !needsForce
		}),

		// Group 4: Final confirmation with dynamic summary
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					resolvedModule := resolveModulePath(appName, modulePath, currentModule)
					resolvedPort := basePort
					if resolvedPort == "" {
						resolvedPort = defaultPort
					}
					featureDesc := describeFeatures(features)
					return fmt.Sprintf("Proceed with app %q, module %s, port %s, features: %s?",
						strings.TrimSpace(appName), resolvedModule, resolvedPort, featureDesc)
				}, &appName).
				Value(&confirm),
		).WithHideFunc(func() bool {
			return needsForce && !force
		}),
	)

	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
		return nil, err
	}

	if needsForce && !force {
		fmt.Println("Setup cancelled.")
		return nil, nil
	}
	if !confirm {
		fmt.Println("Setup cancelled.")
		return nil, nil
	}

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

	// Enforce feature dependencies
	features = enforceFeatureDeps(features)

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
