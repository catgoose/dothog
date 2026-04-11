// setup:feature:demo

package assets

import (
	"os"
	"strings"
	"testing"
)

// TestTavernBundleContract verifies that the committed tavern.min.js bundle
// matches the capabilities required by the templates. This catches silent
// regressions where the JS asset reverts to an older version during branch
// operations or squash merges.
func TestTavernBundleContract(t *testing.T) {
	data, err := os.ReadFile("public/js/tavern.min.js")
	if err != nil {
		t.Fatalf("cannot read tavern.min.js: %v", err)
	}
	src := string(data)

	// Must NOT contain legacy attribute names.
	if strings.Contains(src, "data-tavern-") {
		t.Error("tavern.min.js still reads data-tavern-* attributes; expected tavern-* (v0.0.14+)")
	}

	// Must contain Tavern.command (v0.0.12+).
	if !strings.Contains(src, "command") {
		t.Error("tavern.min.js missing Tavern.command support; expected v0.0.12+")
	}

	// Must contain delegated command support (v0.0.17+).
	if !strings.Contains(src, "commandDelegate") || !strings.Contains(src, "commandTarget") {
		t.Error("tavern.min.js missing delegated command support (commandDelegate/commandTarget); expected v0.0.17+")
	}

	// Must contain tavern-hearth shorthand (v0.0.22+).
	if !strings.Contains(src, "tavern-hearth") {
		t.Error("tavern.min.js missing tavern-hearth shorthand; expected v0.0.22+")
	}

	// Must contain hot-policy support (v0.0.17+).
	if !strings.Contains(src, "hotPolicy") {
		t.Error("tavern.min.js missing hot-policy support; expected v0.0.17+")
	}

	// Must contain stale/live region-state primitives (v0.0.17+).
	if !strings.Contains(src, "staleClass") || !strings.Contains(src, "liveClass") {
		t.Error("tavern.min.js missing stale/live region-state support; expected v0.0.17+")
	}

	// Must contain event names the app dispatches/listens for.
	// These are stable CustomEvent names baked into the bundle (tavern: prefix).
	requiredEvents := []string{
		// Lifecycle (used by Toast Lab, Recovery Lab, Calendar Lab)
		"tavern:disconnected",
		"tavern:reconnected",
		"tavern:live",
		"tavern:recovering",
		"tavern:stale",
		"tavern:replay-gap",
		"tavern:transport-open",
		"tavern:transport-closed",
		// Delegated commands (Hot-Zone Lab, Calendar Lab)
		"tavern:command-sent",
		"tavern:command-success",
		"tavern:command-error",
		// Hot-policy (Hot-Zone Lab, Calendar Lab)
		"tavern:policy-activated",
		"tavern:policy-deactivated",
		// Scoped streams (Toast Lab, Notifications)
		"tavern:stream-warming",
		"tavern:stream-ready",
		"tavern:stream-fallback",
		"tavern:stream-promoted",
		"tavern:stream-retired",
	}
	for _, evt := range requiredEvents {
		if !strings.Contains(src, evt) {
			t.Errorf("tavern.min.js missing event %q; templates depend on this event name", evt)
		}
	}
}
