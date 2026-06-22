package components

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderComponentString(t *testing.T, c templ.Component) string {
	t.Helper()
	var sb strings.Builder
	require.NoError(t, c.Render(context.Background(), &sb))
	return sb.String()
}

func renderWithChildren(t *testing.T, c, child templ.Component) string {
	t.Helper()
	var sb strings.Builder
	require.NoError(t, c.Render(templ.WithChildren(context.Background(), child), &sb))
	return sb.String()
}

func TestTextResourceLink_TraversableAtRest(t *testing.T) {
	out := renderComponentString(t, TextResourceLink("/people/1", "Alpha", "res-link", "font-medium"))
	assert.Contains(t, out, `data-testid="res-link"`)
	assert.Contains(t, out, `class="link font-medium"`, "resource name carries the at-rest link underline")
	assert.NotContains(t, out, "link-hover", "an entity reference never hides its affordance behind hover")
	assert.Contains(t, out, ">Alpha</a>")

	bare := renderComponentString(t, TextResourceLink("/people/1", "Alpha", "", ""))
	assert.Contains(t, bare, `class="link"`)
	assert.NotContains(t, bare, "data-testid")
}

func TestDestinationResourceLink_VisibleLinkNeverButton(t *testing.T) {
	out := renderComponentString(t, DestinationResourceLink("/reports", "Reports", "dest", ""))
	assert.Contains(t, out, "<a ", "a destination stays an anchor")
	assert.Contains(t, out, `data-testid="dest"`)
	assert.Contains(t, out, `class="link link-primary font-medium whitespace-nowrap"`, "destination reads as a visible at-rest link")
	assert.NotContains(t, out, "btn", "a navigation destination is never button-styled")
	assert.NotContains(t, out, "link-hover", "a destination never hides its affordance behind hover")
	assert.Contains(t, out, ">Reports</a>")

	tuned := renderComponentString(t, DestinationResourceLink("/planner", "Planner", "", "text-xs"))
	assert.Contains(t, tuned, `class="link link-primary font-medium whitespace-nowrap text-xs"`)
	assert.NotContains(t, tuned, "data-testid")
}

func TestActionResourceLink_ButtonLikeAnchor(t *testing.T) {
	out := renderComponentString(t, ActionResourceLink("/groups", "View group", "act", "btn-ghost btn-xs"))
	assert.Contains(t, out, "<a ", "navigation stays an anchor")
	assert.Contains(t, out, `class="btn btn-ghost btn-xs"`, "command reads as a button-like action link with exactly one size class")
	assert.NotContains(t, out, "btn-sm", "caller-supplied btn-xs must not collide with a hardcoded btn-sm")
	assert.Contains(t, out, ">View group</a>")
}

func TestActionResourceLink_DefaultsToBtnSm(t *testing.T) {
	out := renderComponentString(t, ActionResourceLink("/groups", "View group", "", ""))
	assert.Contains(t, out, `class="btn btn-sm"`, "empty variant falls back to the neutral btn-sm default")
	assert.NotContains(t, out, "btn-xs")
}

func TestTileResourceLink_RendersAnchorAndChildren(t *testing.T) {
	out := renderWithChildren(t, TileResourceLink("/people", "tile"), templ.Raw("<span>42 people</span>"))
	assert.Contains(t, out, "<a ", "a tile is a clickable anchor")
	assert.Contains(t, out, `data-testid="tile"`)
	assert.Contains(t, out, "<span>42 people</span>", "caller-provided children render inside the anchor")
	assert.Contains(t, out, "</a>")
}

func TestIdentityDisplay_NoNavigation(t *testing.T) {
	c := IdentityChip{Resolved: true, DisplayName: "Bob Builder", Secondary: "bob@example.com"}
	out := renderComponentString(t, IdentityDisplay(c, IdentitySizeBase, true))
	assert.NotContains(t, out, "<a ", "an identity display never navigates")
	assert.Contains(t, out, "Bob Builder")
	assert.Contains(t, out, "bob@example.com")

	noCaption := renderComponentString(t, IdentityDisplay(c, IdentitySizeBase, false))
	assert.NotContains(t, noCaption, "bob@example.com", "showSecondary=false drops the caption")
}

func TestResourceLinks_SanitizeUnsafeHref(t *testing.T) {
	const unsafe = "javascript:alert(1)"

	textOut := renderComponentString(t, TextResourceLink(unsafe, "Alpha", "", ""))
	assert.NotContains(t, textOut, "javascript:", "a caller-supplied unsafe href must not survive as an active scheme")
	assert.Contains(t, textOut, "about:invalid", "templ.URL neutralizes the unsafe scheme")

	c := IdentityChip{Resolved: true, DisplayName: "Alice Agent"}
	idOut := renderComponentString(t, IdentityResourceLink(c, unsafe, "", IdentitySizeBase, false))
	assert.NotContains(t, idOut, "javascript:", "the identity link sanitizes its href the same way")
	assert.Contains(t, idOut, "about:invalid")
}

func TestIdentityResourceLink_LinksNameOnly(t *testing.T) {
	c := IdentityChip{Resolved: true, DisplayName: "Alice Agent", Secondary: "alice@example.com"}
	out := renderComponentString(t, IdentityResourceLink(c, "/people/5", "agent-link", IdentitySizeBase, true))

	assert.Contains(t, out, `data-testid="agent-link"`)
	assert.Contains(t, out, ">Alice Agent</a>", "primary name is the link body")
	assert.Contains(t, out, "alice@example.com")
	assert.Contains(t, out, "text-base-content/50", "secondary caption is muted")
	assert.NotContains(t, out, "link-hover", "the identity name stays traversable at rest")
	assert.Equal(t, 1, strings.Count(out, "<a "), "only the primary name is a link — secondary identity is not its own anchor")

	dense := renderComponentString(t, IdentityResourceLink(c, "/people/5", "agent-link", IdentitySizeCompact, false))
	assert.NotContains(t, dense, "alice@example.com", "showSecondary=false drops the email from dense rows")
}
