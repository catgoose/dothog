// setup:feature:demo

package routes

import (
	"bytes"
	"context"

	"catgoose/dothog/internal/logger"

	"github.com/a-h/templ"
)

// renderToString renders a templ component to an HTML string for use as an
// SSE message body or OOB swap fragment. The description shows up in the
// per-render context log so render failures can be traced back to a call site.
//
// On render error this returns the empty string. Demo routes treat this as
// "skip the publish" — no demo path needs the typed error and surfacing it
// would just clutter every call site.
func renderToString(description string, cmp templ.Component) string {
	buf := &bytes.Buffer{}
	ctx := logger.WithContextIDAndDescription(context.Background(), logger.GenerateContextID(), description)
	if err := cmp.Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}
