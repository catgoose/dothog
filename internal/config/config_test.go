package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTestDefaults ensures the required machine-facing env vars are set for
// non-demo apps. SESSION_SETTINGS_COOKIE_NAME is now strict: tests must
// supply it explicitly because config no longer derives it from APP_NAME.
func setTestDefaults(t *testing.T) {
	t.Helper()
	if os.Getenv("APP_NAME") == "" {
		t.Setenv("APP_NAME", "test-app")
	}
	if os.Getenv("SESSION_SETTINGS_COOKIE_NAME") == "" {
		t.Setenv("SESSION_SETTINGS_COOKIE_NAME", "test_app_session_id")
	}
}

func TestGetConfig(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("SERVER_LISTEN_PORT", "9090")

	config, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "9090", config.ServerPort)

	config2, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, config, config2)
}

func TestGetConfigDefaults(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	// internal/env now distinguishes empty from missing; fall-through to the
	// envStr default only happens when the key is truly unset.
	unsetForTest(t, "SERVER_LISTEN_PORT")
	unsetForTest(t, "DATABASE_URL")

	config, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "3000", config.ServerPort)
	assert.Equal(t, "", config.DatabaseURL)
}

// unsetForTest removes key from the environment for the duration of t and
// restores any prior value on cleanup. t.Setenv only sets values; this helper
// covers the "truly missing" leg required by env.GetDefault's new semantics.
func unsetForTest(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestMustGetConfig(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("SERVER_LISTEN_PORT", "7070")

	config := MustGetConfig()
	assert.Equal(t, "7070", config.ServerPort)
}

func TestConfigEnvOverride(t *testing.T) {
	ResetForTesting()

	t.Setenv("SERVER_LISTEN_PORT", "5555")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("APP_NAME", "testapp")
	t.Setenv("SESSION_SETTINGS_COOKIE_NAME", "custom_session_cookie")

	config, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "5555", config.ServerPort)
	assert.Equal(t, "postgres://localhost/test", config.DatabaseURL)
	assert.Equal(t, "testapp", config.AppName)
	assert.Equal(t, "custom_session_cookie", config.SessionSettingsCookieName)
}

func TestConfigSingleton(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("SERVER_LISTEN_PORT", "1234")

	config1, err := GetConfig()
	require.NoError(t, err)

	config2, err := GetConfig()
	require.NoError(t, err)

	config3 := MustGetConfig()

	assert.Equal(t, config1, config2)
	assert.Equal(t, config1, config3)
}

func TestConfigRequiresSessionSettingsCookieName(t *testing.T) {
	ResetForTesting()

	t.Setenv("APP_NAME", "Test App")
	t.Setenv("SESSION_SETTINGS_COOKIE_NAME", "")

	_, err := GetConfig()
	require.Error(t, err, "config load should fail when SESSION_SETTINGS_COOKIE_NAME is required but unset")
	assert.Contains(t, err.Error(), "SESSION_SETTINGS_COOKIE_NAME")
}

func TestConfigFailsOnMalformedBool(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("CSRF_ROTATE_PER_REQUEST", "not-a-bool")

	_, err := GetConfig()
	require.Error(t, err, "config load should fail when CSRF_ROTATE_PER_REQUEST is malformed")
	assert.Contains(t, err.Error(), "CSRF_ROTATE_PER_REQUEST")
	assert.Contains(t, err.Error(), "not-a-bool")
}

func TestConfigFailsOnMalformedInt(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("GRAPH_USERCACHE_REFRESH_HOUR", "twenty-three")

	_, err := GetConfig()
	require.Error(t, err, "config load should fail when GRAPH_USERCACHE_REFRESH_HOUR is malformed")
	assert.Contains(t, err.Error(), "GRAPH_USERCACHE_REFRESH_HOUR")
}

func TestConfigAcceptsValidBoolInt(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("CSRF_ROTATE_PER_REQUEST", "true")
	t.Setenv("GRAPH_USERCACHE_REFRESH_HOUR", "23")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.True(t, cfg.CSRFRotatePerRequest)
	assert.Equal(t, 23, cfg.GraphUserCacheRefreshHour)
}

func TestConfigUsesDefaultsWhenBoolIntUnset(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("CSRF_ROTATE_PER_REQUEST", "")
	t.Setenv("GRAPH_USERCACHE_REFRESH_HOUR", "")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.False(t, cfg.CSRFRotatePerRequest)
	assert.Equal(t, 5, cfg.GraphUserCacheRefreshHour)
}

func TestConfigRejectsInvalidSessionSettingsCookieName(t *testing.T) {
	ResetForTesting()

	t.Setenv("APP_NAME", "Test App")
	t.Setenv("SESSION_SETTINGS_COOKIE_NAME", "Test App Session")

	_, err := GetConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SESSION_SETTINGS_COOKIE_NAME")
}

func TestConfigAuthOffWhenOIDCAbsent(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("OIDC_ISSUER_URL", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_CLIENT_SECRET", "")
	t.Setenv("OIDC_REDIRECT_URL", "")
	t.Setenv("SESSION_SECRET", "")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.False(t, cfg.AuthConfigured(), "auth should stay off when no OIDC env is set")
}

// TestConfigAuthOffWhenPartialOIDC pins the policy that partial OIDC config
// must not enable auth and must not fail config load. Until the derived app
// finishes setup, auth stays off without surprising startup errors.
func TestConfigAuthOffWhenPartialOIDC(t *testing.T) {
	partials := []struct {
		env  map[string]string
		name string
	}{
		{
			name: "only issuer set",
			env: map[string]string{
				"OIDC_ISSUER_URL": "https://example/issuer",
			},
		},
		{
			name: "issuer + client id, no client secret",
			env: map[string]string{
				"OIDC_ISSUER_URL": "https://example/issuer",
				"OIDC_CLIENT_ID":  "client-x",
			},
		},
		{
			name: "issuer + client id + client secret, no redirect URL",
			env: map[string]string{
				"OIDC_ISSUER_URL":    "https://example/issuer",
				"OIDC_CLIENT_ID":     "client-x",
				"OIDC_CLIENT_SECRET": "shh",
			},
		},
		{
			name: "everything but SESSION_SECRET",
			env: map[string]string{
				"OIDC_ISSUER_URL":    "https://example/issuer",
				"OIDC_CLIENT_ID":     "client-x",
				"OIDC_CLIENT_SECRET": "shh",
				"OIDC_REDIRECT_URL":  "https://example/cb",
			},
		},
	}

	for _, tc := range partials {
		t.Run(tc.name, func(t *testing.T) {
			ResetForTesting()
			setTestDefaults(t)

			// Force each env var off, then layer the partial set back on.
			for _, key := range []string{"OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL", "SESSION_SECRET"} {
				t.Setenv(key, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			cfg, err := GetConfig()
			require.NoError(t, err, "partial OIDC config must not fail config load")
			assert.False(t, cfg.AuthConfigured(), "partial OIDC config must not enable auth")
		})
	}
}

func TestConfigAuthOnWhenCompleteSetupPresent(t *testing.T) {
	ResetForTesting()
	setTestDefaults(t)

	t.Setenv("OIDC_ISSUER_URL", "https://example/issuer")
	t.Setenv("OIDC_CLIENT_ID", "client-x")
	t.Setenv("OIDC_CLIENT_SECRET", "shh")
	t.Setenv("OIDC_REDIRECT_URL", "https://example/cb")
	t.Setenv("SESSION_SECRET", "long-enough-session-secret")

	cfg, err := GetConfig()
	require.NoError(t, err)
	assert.True(t, cfg.AuthConfigured(), "auth should turn on once the full prerequisite set is present")
	assert.Equal(t, "https://example/issuer", cfg.OIDCIssuerURL)
	assert.Equal(t, "client-x", cfg.OIDCClientID)
	assert.Equal(t, "long-enough-session-secret", cfg.SessionSecret)
}
