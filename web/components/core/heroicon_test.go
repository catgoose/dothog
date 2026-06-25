package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeroIcon_NamedIconRendersSVGPath(t *testing.T) {
	out := renderComponentString(t, heroIcon("check"))
	assert.Contains(t, out, "<svg")
	assert.Contains(t, out, `d="M4.5 12.75l6 6 9-13.5"`)
	assert.NotContains(t, out, "icon-")
}

func TestHeroIcon_PathDataRendersSVGPath(t *testing.T) {
	// Shared nav items pass Heroicon path data directly as the icon name.
	const navPath = "M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3"
	out := renderComponentString(t, heroIcon(navPath))
	assert.Contains(t, out, "<svg")
	assert.Contains(t, out, `d="`+navPath+`"`)
}

func TestHeroIcon_EmptyRendersNothing(t *testing.T) {
	assert.Empty(t, renderComponentString(t, heroIcon("")))
}
