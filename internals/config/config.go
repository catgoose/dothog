// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"strconv"
	"sync"

	"catgoose/go-htmx-demo/internals/logger"

	// setup:feature:auth:start
	"github.com/catgoose/crooner"
	// setup:feature:auth:end
	"github.com/catgoose/dio"
)

// AppConfig holds application configuration values
type AppConfig struct {
	// setup:feature:auth:start
	SessionMgr    crooner.SessionManager
	CroonerConfig *crooner.AuthConfigParams
	SessionSecret string
	AppName       string
	// setup:feature:auth:end
	ServerPort string
	// setup:feature:graph:start
	AzureRefreshUsersHour int
	// setup:feature:graph:end
	// setup:feature:database:start
	// EnableDatabase controls whether the MSSQL database is initialised on startup.
	// Set to false (the default) when running the template without a database configured.
	// Set to true (via ENABLE_DATABASE=true) when a database is available.
	EnableDatabase bool
	// InitRepo, when true, runs schema init on startup. This is destructive: it drops and recreates tables and wipes existing data. Set false in production or when preserving data.
	InitRepo bool
	// setup:feature:database:end
	// setup:feature:auth:start
	CroonerDisabled bool
	// setup:feature:auth:end
}

// getEnvVar safely retrieves an environment variable and logs errors consistently
func getEnvVar(key string, description string) (string, error) {
	value, err := dio.Env(key)
	if err != nil {
		return "", logger.LogAndReturnError(fmt.Sprintf("Failed to get %s", description), err)
	}
	return value, nil
}

func buildConfig() (*AppConfig, error) {
	port, err := getEnvVar("SERVER_LISTEN_PORT", "server listen port")
	if err != nil {
		return nil, fmt.Errorf("error getting server listen port: %w", err)
	}

	// setup:feature:database:start
	// Database configuration (disabled by default so the template runs without a DB)
	enableDatabase := false
	if v, err := dio.Env("ENABLE_DATABASE"); err == nil {
		if parsed, err := strconv.ParseBool(v); err == nil {
			enableDatabase = parsed
		}
	}
	// setup:feature:database:end

	// setup:feature:graph:start
	// Azure user refresh hour configuration (default: 5 AM)
	refreshHour := 5
	if refreshHourStr, err := dio.Env("AZURE_USER_REFRESH_HOUR"); err == nil {
		if h, err := strconv.Atoi(refreshHourStr); err == nil && h >= 0 && h < 24 {
			refreshHour = h
		} else {
			return nil, fmt.Errorf("invalid AZURE_USER_REFRESH_HOUR value: %q (must be integer 0-23)", refreshHourStr)
		}
	}
	// setup:feature:graph:end

	// setup:feature:auth:start
	croonerDisabled := true
	var croonerConfig *crooner.AuthConfigParams
	var sessionSecret, appName string
	if !croonerDisabled {
		azureClientID, err := getEnvVar("AZURE_CLIENT_ID", "azure client id")
		if err != nil {
			return nil, err
		}
		azureClientSecret, err := getEnvVar("AZURE_CLIENT_SECRET", "azure client secret")
		if err != nil {
			return nil, err
		}
		azureTenantID, err := getEnvVar("AZURE_TENANT_ID", "azure tenant id")
		if err != nil {
			return nil, err
		}
		redirectURL, err := getEnvVar("AZURE_REDIRECT_URL", "azure redirect url")
		if err != nil {
			return nil, err
		}
		logoutURLRedirect, err := getEnvVar("AZURE_LOGOUT_REDIRECT_URL", "azure logout redirect url")
		if err != nil {
			return nil, err
		}
		loginURLRedirect, err := getEnvVar("AZURE_LOGIN_REDIRECT_URL", "azure login redirect url")
		if err != nil {
			return nil, err
		}
		sessionSecret, err = getEnvVar("SESSION_SECRET", "session secret")
		if err != nil {
			return nil, err
		}
		appName = "app"
		if name, err := dio.Env("APP_NAME"); err == nil && name != "" {
			appName = name
		}
		croonerConfig = &crooner.AuthConfigParams{
			ClientID:          azureClientID,
			ClientSecret:      azureClientSecret,
			TenantID:          azureTenantID,
			RedirectURL:       redirectURL,
			LogoutURLRedirect: logoutURLRedirect,
			LoginURLRedirect:  loginURLRedirect,
			AuthRoutes: &crooner.AuthRoutes{
				Login:    "/login",
				Logout:   "/logout",
				Callback: "/callback",
			},
			SessionValueClaims: []map[string]string{
				{"azureId": "oid"},
				{"groups": "roles"},
			},
			SecurityHeaders: &crooner.SecurityHeadersConfig{
				ContentSecurityPolicy: "img-src 'self' data: https://login.microsoftonline.com;",
			},
		}
	}
	// setup:feature:auth:end

	return &AppConfig{
		ServerPort: port,
		// setup:feature:graph:start
		AzureRefreshUsersHour: refreshHour,
		// setup:feature:graph:end
		// setup:feature:database:start
		EnableDatabase: enableDatabase,
		// setup:feature:database:end
		// setup:feature:auth:start
		CroonerDisabled: croonerDisabled,
		CroonerConfig:   croonerConfig,
		SessionSecret:   sessionSecret,
		AppName:         appName,
		// setup:feature:auth:end
		// setup:feature:database:start
		InitRepo: false,
		// setup:feature:database:end
	}, nil
}

var getConfig = sync.OnceValues(buildConfig)

// GetConfig returns the singleton configuration instance.
// The config is built on first call and cached for all subsequent calls.
func GetConfig() (*AppConfig, error) {
	return getConfig()
}

// MustGetConfig returns the singleton configuration instance.
// It panics if the config cannot be loaded.
func MustGetConfig() *AppConfig {
	config, err := GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
	return config
}

// ResetForTesting resets the singleton instance for testing purposes.
// This should only be used in tests.
func ResetForTesting() {
	getConfig = sync.OnceValues(buildConfig)
}
