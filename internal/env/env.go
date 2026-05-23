// Package env loads environment variables from .env.{mode} files and exposes
// the current mode via predicates. Mode resolution follows: the explicit -env
// flag, then the ENV env var, then "development".
package env

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// envFlag overrides ENV and the default when set. Empty string means "not
// provided" — godot helpers can't disambiguate "user passed -env=" from
// "user did not pass -env", so the sentinel is the empty string and the
// fallback chain handles the rest.
var envFlag = flag.String("env", "", "application environment (reads .env.{mode}); falls back to ENV, then \"development\"")

var mode string

// Init resolves the runtime mode using the precedence: -env flag, then ENV
// env var, then "development". It then loads .env.{mode} via godotenv; a
// missing file errors, but callers may proceed with the OS env.
func Init() error {
	mode = resolveMode()
	file := fmt.Sprintf(".env.%s", mode)
	if err := godotenv.Load(file); err != nil {
		return fmt.Errorf("env file not found: %s: %w", file, err)
	}
	return nil
}

func resolveMode() string {
	if envFlag != nil && *envFlag != "" {
		return normalize(*envFlag)
	}
	if v := os.Getenv("ENV"); v != "" {
		return normalize(v)
	}
	return "development"
}

// Lookup returns the raw OS environment value and a presence flag. Empty
// string is preserved (an explicitly empty env var is distinct from a
// missing one).
func Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Get returns the value of key. The error reports only "not set"; an
// explicitly empty value returns ("", nil) so callers can distinguish it
// from a missing key.
func Get(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("env %s not set", key)
	}
	return v, nil
}

// GetDefault returns key's value when set (even if empty), or fallback when
// the key is not set. An explicitly empty env var returns "" — fallback
// applies only to truly missing keys.
func GetDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// Dev reports whether the current mode is "development".
func Dev() bool { return mode == "development" }

// Name is the canonical mode string ("development", "production", …) after normalisation.
func Name() string { return mode }

func normalize(s string) string {
	switch strings.ToLower(s) {
	case "dev":
		return "development"
	case "prod":
		return "production"
	default:
		return strings.ToLower(s)
	}
}
