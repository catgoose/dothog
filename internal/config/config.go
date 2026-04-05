// Package config provides configuration management for the application.
// AppConfig is a flat struct read once at startup via GetConfig(). Each field
// has a sensible Go default that can be overridden by environment variables.
// Derived apps extend AppConfig with additional fields and read additional
// env vars in buildConfig(). The setup wizard controls which fields exist
// via feature gate comments.
package config

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"catgoose/dothog/internal/logger"

	// setup:feature:auth:start
	"github.com/catgoose/crooner"
	// setup:feature:auth:end
	appenv "catgoose/dothog/internal/env"
)

// AppConfig holds all application configuration. Flat struct, globally
// accessible via GetConfig()/MustGetConfig(). Extend by adding fields
// and reading them in buildConfig().
type AppConfig struct {
	// setup:feature:auth:start
	SessionMgr    crooner.SessionManager
	CroonerConfig *crooner.AuthConfigParams
	// setup:feature:auth:end
	ServerPort                string
	DatabaseURL               string
	SessionSecret             string
	AppName                   string
	CSRFPerRequestPaths       []string
	CSRFExemptPaths           []string
	GraphUserCacheRefreshHour int
	// setup:feature:auth:start
	CroonerDisabled bool
	// setup:feature:auth:end
	EnableDatabase       bool
	InitRepo             bool
	CSRFRotatePerRequest bool
}

func buildConfig() (*AppConfig, error) {
	cfg := &AppConfig{
		// Defaults — override with env vars
		ServerPort:  envStr("SERVER_LISTEN_PORT", "3000"),
		AppName:     envStr("APP_NAME", ""),
		DatabaseURL: envStr("DATABASE_URL", "sqlite:///db/app.db"),
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

	// setup:feature:database:start
	cfg.EnableDatabase = envBool("ENABLE_DATABASE", false)
	// setup:feature:database:end

	// setup:feature:auth:start
	cfg.CroonerDisabled = true
	issuerURL := envStr("OIDC_ISSUER_URL", "")
	clientID := envStr("OIDC_CLIENT_ID", "")
	if issuerURL != "" && clientID != "" {
		cfg.CroonerDisabled = false
		secret, err := getEnvVar("SESSION_SECRET", "session secret")
		if err != nil {
			return nil, err
		}
		cfg.SessionSecret = secret
		cfg.CroonerConfig = &crooner.AuthConfigParams{
			IssuerURL:         issuerURL,
			ClientID:          clientID,
			ClientSecret:      envStr("OIDC_CLIENT_SECRET", ""),
			RedirectURL:       envStr("OIDC_REDIRECT_URL", ""),
			LogoutURLRedirect: envStr("OIDC_LOGOUT_REDIRECT_URL", "/"),
			LoginURLRedirect:  envStr("OIDC_LOGIN_REDIRECT_URL", "/"),
			AuthRoutes: &crooner.AuthRoutes{
				Login:    "/login",
				Logout:   "/logout",
				Callback: "/callback",
			},
		}
	}
	// setup:feature:auth:end

	// setup:feature:csrf:start
	cfg.CSRFRotatePerRequest = envBool("CSRF_ROTATE_PER_REQUEST", false)
	cfg.CSRFPerRequestPaths = envList("CSRF_PER_REQUEST_PATHS")
	cfg.CSRFExemptPaths = []string{"/login", "/callback", "/logout", "/report-issue"}
	// setup:feature:csrf:end

	// setup:feature:graph:start
	cfg.GraphUserCacheRefreshHour = envInt("GRAPH_USERCACHE_REFRESH_HOUR", 5)
	// setup:feature:graph:end

	return cfg, nil
}

// --- Env helpers ---

func envStr(key, fallback string) string {
	return appenv.GetDefault(key, fallback)
}

func envBool(key string, fallback bool) bool {
	if v, err := appenv.Get(key); err == nil {
		if parsed, err := strconv.ParseBool(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := appenv.Get(key); err == nil {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
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

func getEnvVar(key string, description string) (string, error) {
	value, err := appenv.Get(key)
	if err != nil {
		return "", logger.LogAndReturnError(fmt.Sprintf("Failed to get %s", description), err)
	}
	return value, nil
}

// --- Singleton ---

var getConfig = sync.OnceValues(buildConfig)

// GetConfig returns the singleton configuration instance.
func GetConfig() (*AppConfig, error) {
	return getConfig()
}

// MustGetConfig returns the singleton configuration instance.
// Panics if configuration cannot be loaded.
func MustGetConfig() *AppConfig {
	config, err := GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	return config
}

// ResetForTesting resets the singleton for testing purposes.
func ResetForTesting() {
	getConfig = sync.OnceValues(buildConfig)
}
