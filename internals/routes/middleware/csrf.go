// setup:feature:csrf
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/catgoose/crooner"
	"github.com/labstack/echo/v4"
)

const csrfTokenSessionKey = "csrf_token"
const csrfContextKey = "csrf_token"

var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

// CSRFConfig holds CSRF middleware configuration.
type CSRFConfig struct {
	PerRequestPaths  []string
	ExemptPaths      []string
	RotatePerRequest bool
}

// CSRF returns Echo middleware that generates and validates CSRF tokens using the session.
// When sm is nil the middleware is a no-op. Requires crooner session (CSRF requires crooner).
func CSRF(sm crooner.SessionManager, cfg CSRFConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if sm == nil {
				return next(c)
			}
			path := c.Request().URL.Path
			if pathExempt(cfg.ExemptPaths, path) {
				return next(c)
			}
			if safeMethods[c.Request().Method] {
				token, err := getOrCreateToken(c, sm, cfg, path)
				if err != nil {
					return err
				}
				c.Set(csrfContextKey, token)
				return next(c)
			}
			reqToken := c.Request().Header.Get("X-CSRF-Token")
			if reqToken == "" {
				reqToken = c.Request().FormValue("_csrf")
			}
			sessionToken, err := crooner.GetString(sm, c, csrfTokenSessionKey)
			if err != nil || sessionToken == "" || !equalConstantTime(sessionToken, reqToken) {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

func pathExempt(exempt []string, path string) bool {
	for _, e := range exempt {
		if path == e || strings.HasPrefix(path, strings.TrimSuffix(e, "/")+"/") {
			return true
		}
	}
	return false
}

func pathPerRequest(paths []string, path string) bool {
	for _, p := range paths {
		if path == p {
			return true
		}
	}
	return false
}

func getOrCreateToken(c echo.Context, sm crooner.SessionManager, cfg CSRFConfig, path string) (string, error) {
	rotate := cfg.RotatePerRequest || pathPerRequest(cfg.PerRequestPaths, path)
	if !rotate {
		existing, err := crooner.GetString(sm, c, csrfTokenSessionKey)
		if err == nil && existing != "" {
			return existing, nil
		}
	}
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	if err := sm.Set(c, csrfTokenSessionKey, token); err != nil {
		return "", err
	}
	return token, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func equalConstantTime(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
