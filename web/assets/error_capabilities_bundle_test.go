package assets

import (
	"os"
	"strings"
	"testing"
)

// TestErrorCapabilitiesBundleContract pins the JS-side hook that injects
// X-Error-Accept-Surfaces / X-Error-Fallback-Surface so the negotiation in
// internal/routes/handler/error_capabilities.go has a wire counterpart. The
// server parses these headers and degrades surfaces accordingly; a silent
// regression in the JS file would break that contract without surfacing in
// Go tests.
func TestErrorCapabilitiesBundleContract(t *testing.T) {
	data, err := os.ReadFile("public/js/htmx.error-capabilities.js")
	if err != nil {
		t.Fatalf("cannot read htmx.error-capabilities.js: %v", err)
	}
	src := string(data)

	for _, marker := range []string{
		"htmx:configRequest",
		"data-error-accept",
		"data-error-fallback",
		"X-Error-Accept-Surfaces",
		"X-Error-Fallback-Surface",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("htmx.error-capabilities.js missing required marker %q — JS/server contract drift", marker)
		}
	}
}
