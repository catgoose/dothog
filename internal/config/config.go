// Package config loads immutable, env-backed application values at startup.
// AppConfig is a flat value object — no runtime-mutated state, no constructed
// session managers, no crooner types. Auth/session/CSRF runtime is built
// locally in InitEcho from the raw env values exposed here.
//
// Derived apps extend AppConfig with additional fields and read additional
// env vars in buildConfig(). The setup wizard controls which fields exist
// via feature gate comments.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	// setup:feature:session_settings:start
	"catgoose/dothog/internal/session"
	// setup:feature:session_settings:end
	appenv "catgoose/dothog/internal/env"
)

// AppConfig holds all application configuration. Flat struct, globally
// accessible via GetConfig()/MustGetConfig(). Extend by adding fields
// and reading them in buildConfig(). Values are read once and never mutated
// after load.
type AppConfig struct {
	ServerPort    string
	DatabaseURL   string
	SessionSecret string
	AppName       string
	// setup:feature:session_settings:start
	SessionSettingsCookieName string
	// setup:feature:session_settings:end
	// setup:feature:auth:start
	OIDCIssuerURL         string
	OIDCClientID          string
	OIDCClientSecret      string
	OIDCRedirectURL       string
	OIDCLoginRedirectURL  string
	OIDCLogoutRedirectURL string
	// setup:feature:auth:end
	CSRFPerRequestPaths       []string
	CSRFExemptPaths           []string
	GraphUserCacheRefreshHour int
	CSRFRotatePerRequest      bool
	// ContentSecurityPolicy carries the verbatim header value to emit on
	// every response when non-empty. Setup writes a strict policy into
	// CSP_HEADER for derived apps that select the csp feature; bare source
	// runs and scaffolds that did not opt in leave the field empty so no
	// header is set. The runtime contract is config-driven so demo-bearing
	// builds (source repo, demo scaffold) cannot accidentally claim CSP.
	ContentSecurityPolicy string
}

// setup:feature:auth:start

// AuthConfigured reports whether the derived app has supplied a complete
// auth-ready environment. Partial OIDC config (e.g. issuer set but client ID
// missing) returns false so auth simply stays off rather than half-booting or
// failing config load. Routes consult this to decide whether to bootstrap
// crooner; setup-incomplete scaffolds run with auth disabled.
func (c *AppConfig) AuthConfigured() bool {
	return c.OIDCIssuerURL != "" &&
		c.OIDCClientID != "" &&
		c.OIDCClientSecret != "" &&
		c.OIDCRedirectURL != "" &&
		c.SessionSecret != ""
}

// setup:feature:auth:end

func buildConfig() (*AppConfig, error) {
	cfg := &AppConfig{
		ServerPort:  envStr("SERVER_LISTEN_PORT", "3000"),
		AppName:     envStr("APP_NAME", ""),
		DatabaseURL: envStr("DATABASE_URL", ""),
	}

	// APP_NAME: required unless demo provides a fallback
	// setup:feature:demo:start
	if cfg.AppName == "" {
		cfg.AppName = "dothog"
	}
	// setup:feature:demo:end
	if cfg.AppName == "" {
		return nil, fmt.Errorf("APP_NAME is required")
	}

	// setup:feature:session_settings:start
	// SESSION_SETTINGS_COOKIE_NAME is machine-facing: required, not derived.
	// Deriving from APP_NAME (a presentation label) would silently rebind
	// sessions if the app name ever changed. Setup writes a stable default
	// into .env.development; deployments must set the value explicitly.
	cookieName := envStr("SESSION_SETTINGS_COOKIE_NAME", "")
	if cookieName == "" {
		return nil, fmt.Errorf("SESSION_SETTINGS_COOKIE_NAME is required when session_settings is enabled")
	}
	if !session.IsValidCookieName(cookieName) {
		return nil, fmt.Errorf("SESSION_SETTINGS_COOKIE_NAME contains invalid cookie-name characters")
	}
	cfg.SessionSettingsCookieName = cookieName
	// setup:feature:session_settings:end

	// setup:feature:auth:start
	// Read every auth env value into the immutable AppConfig. Missing values
	// just leave the corresponding field empty; AuthConfigured() flips on only
	// when the full set is present, so partial setups run with auth off
	// instead of failing config load.
	cfg.OIDCIssuerURL = envStr("OIDC_ISSUER_URL", "")
	cfg.OIDCClientID = envStr("OIDC_CLIENT_ID", "")
	cfg.OIDCClientSecret = envStr("OIDC_CLIENT_SECRET", "")
	cfg.OIDCRedirectURL = envStr("OIDC_REDIRECT_URL", "")
	cfg.OIDCLoginRedirectURL = envStr("OIDC_LOGIN_REDIRECT_URL", "/")
	cfg.OIDCLogoutRedirectURL = envStr("OIDC_LOGOUT_REDIRECT_URL", "/")
	cfg.SessionSecret = envStr("SESSION_SECRET", "")
	// setup:feature:auth:end

	// setup:feature:csrf:start
	csrfRotate, err := envBool("CSRF_ROTATE_PER_REQUEST", false)
	if err != nil {
		return nil, err
	}
	cfg.CSRFRotatePerRequest = csrfRotate
	cfg.CSRFPerRequestPaths = envList("CSRF_PER_REQUEST_PATHS")
	cfg.CSRFExemptPaths = []string{"/login", "/callback", "/logout", "/report-issue"}
	// setup:feature:csrf:end

	// setup:feature:graph:start
	graphHour, err := envInt("GRAPH_USERCACHE_REFRESH_HOUR", 5)
	if err != nil {
		return nil, err
	}
	cfg.GraphUserCacheRefreshHour = graphHour
	// setup:feature:graph:end

	cfg.ContentSecurityPolicy = envStr("CSP_HEADER", "")

	return cfg, nil
}

// --- Env helpers ---

func envStr(key, fallback string) string {
	return appenv.GetDefault(key, fallback)
}

// envBool reads key as a strconv.ParseBool value. An unset/empty key uses
// fallback; a present-but-malformed value fails config load rather than
// silently falling back, so misconfigured deployments surface at startup
// instead of running with surprise defaults.
func envBool(key string, fallback bool) (bool, error) {
	v, err := appenv.Get(key)
	if err != nil || v == "" {
		return fallback, nil
	}
	parsed, perr := strconv.ParseBool(v)
	if perr != nil {
		return false, fmt.Errorf("%s must be a valid boolean (got %q): %w", key, v, perr)
	}
	return parsed, nil
}

// envInt reads key as a strconv.Atoi value. An unset/empty key uses
// fallback; a present-but-malformed value fails config load rather than
// silently falling back.
func envInt(key string, fallback int) (int, error) {
	v, err := appenv.Get(key)
	if err != nil || v == "" {
		return fallback, nil
	}
	parsed, perr := strconv.Atoi(v)
	if perr != nil {
		return 0, fmt.Errorf("%s must be a valid integer (got %q): %w", key, v, perr)
	}
	return parsed, nil
}

func envList(key string) []string {
	v, err := appenv.Get(key)
	if err != nil || v == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}

// --- Singleton ---

var getConfig = sync.OnceValues(buildConfig)

// GetConfig is the sync.OnceValues-cached AppConfig; first call builds, later calls reuse.
func GetConfig() (*AppConfig, error) {
	return getConfig()
}

// MustGetConfig is the singleton AppConfig; panics on load failure (use only at startup).
func MustGetConfig() *AppConfig {
	config, err := GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	return config
}

// ResetForTesting clears the cached config singleton so the next GetConfig rebuilds it; tests only.
func ResetForTesting() {
	getConfig = sync.OnceValues(buildConfig)
}
