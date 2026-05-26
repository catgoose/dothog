package assets

import (
	"os"
	"strings"
	"testing"
)

// TestErrorCapabilitiesBundleContract verifies that the HTMX-side hook still
// emits the request headers consumed by error-capability negotiation on the
// server.
func TestErrorCapabilitiesBundleContract(t *testing.T) {
	data, err := os.ReadFile("public/js/app/htmx.error-capabilities.js")
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
