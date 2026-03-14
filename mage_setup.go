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

	"github.com/magefile/mage/sh"
)

const templateModulePath = "catgoose/dothog"

// featureLabels maps feature tags to human-readable gum labels.
var featureLabels = map[string]string{
	setup.FeatureAuth:     "Auth (Crooner)",
	setup.FeatureGraph:    "Graph API",
	setup.FeatureDatabase: "Database (MSSQL)",
	setup.FeatureSSE:      "SSE (requires Caddy)",
	setup.FeatureCaddy:    "Caddy (HTTPS)",
	setup.FeatureAvatar:   "Avatar Photos (requires Graph)",
	setup.FeatureDemo:             "Demo Content",
	setup.FeatureSessionSettings:  "Session Settings (SQLite)",
}

// featureLabelOrder is the display order for the gum multi-select.
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
		cleanup, err := gumConfirm("Cleanup template files and setup helpers from this repo?")
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

	if !hasGum() {
		return errors.New("APP_NAME is required; run with -n APP_NAME or install gum for the wizard")
	}

	copyFirst, err := gumConfirm("Copy template to a new directory before setting up?")
	if err != nil {
		return err
	}
	if copyFirst {
		target, err := gumInput("Target directory: ", "e.g. ../my-app or /path/to/project", "")
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
				ok, _ := gumConfirm("Target directory exists and is not empty. Overwrite?")
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
		gitInit, _ := gumConfirm("Run git init in the new directory?")
		if gitInit {
			cmd := exec.Command("git", "init")
			cmd.Dir = absTarget
			_ = cmd.Run()
		}
		if err := setup.Run(context.Background(), absTarget, *opts); err != nil {
			return err
		}
		_ = os.RemoveAll(filepath.Join(absTarget, setup.TemplateSetupDir))
		_ = os.RemoveAll(filepath.Join(absTarget, "internal", "setup"))
		_ = os.Remove(filepath.Join(absTarget, "mage_setup.go"))
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
	cleanup, err := gumConfirm("Cleanup template files and setup helpers from this repo?")
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
	appName, err := gumInput("App name: ", "App name (e.g. My App)", "")
	if err != nil {
		return nil, err
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return nil, errors.New("APP_NAME is required")
	}

	binary := binaryNameFromApp(appName)
	currentModule := goModulePath()

	modulePath := currentModule
	if modulePath == "" {
		modulePath = fmt.Sprintf("github.com/you/%s", binary)
	} else if modulePath == templateModulePath {
		modulePath = fmt.Sprintf("%s-%s", templateModulePath, binary)
	}

	modulePathInput, err := gumInputWithValue(
		"Module path: ",
		"Module path (e.g. github.com/you/my-app)",
		"",
	)
	if err != nil {
		return nil, err
	}
	modulePathInput = strings.TrimSpace(modulePathInput)
	if modulePathInput != "" {
		modulePath = modulePathInput
	}

	defaultPort := randomBasePort()
	basePortStr := fmt.Sprintf("%d", defaultPort)

	inputBasePort, err := gumInputWithValue(
		"Base port: ",
		fmt.Sprintf("5-digit base port < 60000 (blank = %s)", basePortStr),
		basePortStr,
	)
	if err != nil {
		return nil, err
	}
	inputBasePort = strings.TrimSpace(inputBasePort)
	if inputBasePort != "" {
		basePortStr = inputBasePort
	}

	if len(basePortStr) != 5 {
		return nil, fmt.Errorf("BASE_PORT must be a 5-digit number, got: %s", basePortStr)
	}
	var basePortNum int
	if _, err := fmt.Sscanf(basePortStr, "%d", &basePortNum); err != nil || basePortNum >= 60000 {
		return nil, fmt.Errorf("BASE_PORT must be < 60000, got: %s", basePortStr)
	}

	// Feature selection
	features, err := wizardFeatureSelect()
	if err != nil {
		return nil, err
	}
	if features == nil {
		fmt.Println("Setup cancelled.")
		return nil, nil
	}

	force := false
	if currentModule != "" && currentModule != templateModulePath {
		msg := fmt.Sprintf("Module already customized (go.mod: %s). Run setup again with --force?", currentModule)
		ok, err := gumConfirm(msg)
		if err != nil {
			return nil, err
		}
		if !ok {
			fmt.Println("Setup cancelled.")
			return nil, nil
		}
		force = true
	}

	featureDesc := "all"
	if len(features) == 0 {
		featureDesc = "none (bare HTMX app)"
	} else if len(features) < len(setup.AllFeatures) {
		featureDesc = strings.Join(features, ", ")
	}

	ok, err := gumConfirm(fmt.Sprintf("Proceed with app %q, module %s, port %s, features: %s?", appName, modulePath, basePortStr, featureDesc))
	if err != nil || !ok {
		fmt.Println("Setup cancelled.")
		return nil, nil
	}

	return &setup.Options{
		AppName:     appName,
		ModulePath:  modulePath,
		BasePort:    basePortStr,
		Force:       force,
		Features:    features,
		ConfirmFunc: gumConfirm,
	}, nil
}

// wizardFeatureSelect presents the interactive feature multi-select.
// Returns the selected feature tags, or nil if the user cancelled.
func wizardFeatureSelect() ([]string, error) {
	// Build ordered label list and label→tag map
	var labels []string
	var preselected []string
	labelToTag := make(map[string]string)
	for _, tag := range featureLabelOrder {
		label := featureLabels[tag]
		labels = append(labels, label)
		labelToTag[label] = tag
		// Demo is opt-in: not preselected by default
		if tag != setup.FeatureDemo {
			preselected = append(preselected, label)
		}
	}

	selected, err := gumChooseMulti("Select features to include (all selected by default):", labels, preselected)
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, nil // user cancelled
	}

	// Map labels back to tags
	var tags []string
	tagSet := make(map[string]bool)
	for _, label := range selected {
		if tag, ok := labelToTag[label]; ok {
			tags = append(tags, tag)
			tagSet[tag] = true
		}
	}

	// SSE requires Caddy — auto-include if missing
	if tagSet[setup.FeatureSSE] && !tagSet[setup.FeatureCaddy] {
		tags = append(tags, setup.FeatureCaddy)
		fmt.Println("SSE requires Caddy for proxying; Caddy auto-included.")
	}

	// Avatar requires Graph — auto-include if missing
	if tagSet[setup.FeatureAvatar] && !tagSet[setup.FeatureGraph] {
		tags = append(tags, setup.FeatureGraph)
		fmt.Println("Avatar requires Graph API; Graph auto-included.")
	}

	return tags, nil
}

func hasFeature(features []string, tag string) bool {
	for _, f := range features {
		if f == tag {
			return true
		}
	}
	return false
}

func gumChooseMulti(header string, items []string, selected []string) ([]string, error) {
	args := []string{"choose", "--no-limit", "--header", header}
	if len(selected) > 0 {
		args = append(args, "--selected", strings.Join(selected, ","))
	}
	args = append(args, items...)
	out, err := sh.Output("gum", args...)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil // user cancelled
		}
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
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
	if hasGum() {
		opts.ConfirmFunc = gumConfirm
	} else {
		opts.ConfirmFunc = stdinConfirm
	}
	return opts, true, false, nil
}

// parseFeatureFlag is defined in magefile.go (survives mage_setup.go removal).

func printSetupUsage() {
	fmt.Println(`Usage: go tool mage setup [-n APP_NAME] [-m MODULE_PATH] [-p BASE_PORT] [--features FEATURES] [--no-caddy] [--force]

  -n APP_NAME        Human-readable app name (e.g. "My App"). Required.
  -m MODULE_PATH     Go module path (e.g. "github.com/you/my-app").
  -p BASE_PORT       5-digit base port < 60000; APP_TLS_PORT=BASE_PORT, TEMPL_HTTP_PORT=BASE_PORT+1, CADDY_TLS_PORT=BASE_PORT+2.
  --features LIST    Comma-separated feature tags to keep: auth,graph,avatar,database,sse,caddy,demo.
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

func hasGum() bool {
	_, err := sh.Exec(nil, nil, nil, "which", "gum")
	return err == nil
}

func gumInput(prompt, placeholder, value string) (string, error) {
	args := []string{"input", "--prompt", prompt}
	if placeholder != "" {
		args = append(args, "--placeholder", placeholder)
	}
	if value != "" {
		args = append(args, "--value", value)
	}
	out, err := sh.Output("gum", args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func gumInputWithValue(prompt, placeholder, value string) (string, error) {
	return gumInput(prompt, placeholder, value)
}

func gumConfirm(message string) (bool, error) {
	err := sh.Run("gum", "confirm", message)
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func stdinConfirm(msg string) (bool, error) {
	fmt.Printf("%s [y/N] ", msg)
	var answer string
	fmt.Scanln(&answer)
	return strings.ToLower(strings.TrimSpace(answer)) == "y", nil
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
