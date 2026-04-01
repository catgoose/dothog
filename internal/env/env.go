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

// Init loads environment variables from the .env.{mode} file. The mode is
// determined by (in order): the env parameter if non-empty, the ENV
// environment variable if set, or the -env flag (default "development").
// Returns an error if the env file is missing; callers may choose to continue
// with OS environment variables.
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

// Get returns the value of the environment variable or an error if not set.
func Get(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("env %s not set", key)
	}
	return v, nil
}

// GetDefault returns the value of the environment variable or fallback if not set.
func GetDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Dev reports whether the current mode is "development".
func Dev() bool { return mode == "development" }

// Name returns the current environment mode.
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
