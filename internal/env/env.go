// Package env loads environment variables from .env.{mode} files and exposes
// the current mode via predicates.
package env

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var envFlag = flag.String("env", "development", "application environment (reads .env.{mode})")

var mode string

// Init resolves the mode from env (or the -env flag, default "development") and loads
// .env.{mode} via godotenv; a missing file errors, but callers may proceed with the OS env.
func Init(env string) error {
	if env == "" {
		env = *envFlag
	}
	mode = normalize(env)
	file := fmt.Sprintf(".env.%s", mode)
	if err := godotenv.Load(file); err != nil {
		return fmt.Errorf("env file not found: %s: %w", file, err)
	}
	return nil
}

// Get errors when key is unset; use GetDefault for optional values.
func Get(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("env %s not set", key)
	}
	return v, nil
}

// GetDefault treats an unset key as fallback; never returns an error.
func GetDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
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
