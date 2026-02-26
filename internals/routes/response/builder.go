// Package response provides a fluent builder for composing HTMX responses
// with out-of-band swaps and HX-* response headers.
package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"catgoose/go-htmx-template/internals/routes/htmx"
	"catgoose/go-htmx-template/internals/routes/hypermedia"
	corecomponents "catgoose/go-htmx-template/web/components/core"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// Builder composes an HTMX response with optional OOB components and HX-* headers.
// Call Send() to flush everything atomically: all components are rendered into a
// buffer before any byte is written to the wire, so a render failure never
// produces a partial response.
type Builder struct {
	c          echo.Context
	primary    templ.Component
	err        error
	headers    map[string]string
	oob        []templ.Component
	statusCode int
}

// New creates a Builder defaulting to HTTP 200.
func New(c echo.Context) *Builder {
	return &Builder{
		c:          c,
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}

// Status overrides the HTTP status code.
func (b *Builder) Status(code int) *Builder {
	b.statusCode = code
	return b
}

// Component sets the primary (inline) templ component for the response.
func (b *Builder) Component(cmp templ.Component) *Builder {
	b.primary = cmp
	return b
}

// OOB appends an out-of-band component. HTMX processes elements with
// hx-swap-oob as OOB swaps; the components must include that attribute.
func (b *Builder) OOB(cmp templ.Component) *Builder {
	b.oob = append(b.oob, cmp)
	return b
}

// OOBErrorStatus sets ec.OOBTarget (defaulting to "#error-status") and appends
// the error panel as an OOB component. The ErrorStatusFromContext component
// will carry hx-swap-oob so HTMX routes it to the target.
func (b *Builder) OOBErrorStatus(ec hypermedia.ErrorContext) *Builder {
	if ec.OOBTarget == "" {
		ec.OOBTarget = hypermedia.DefaultErrorStatusTarget
	}
	if ec.OOBSwap == "" {
		ec.OOBSwap = "innerHTML"
	}
	return b.OOB(corecomponents.ErrorStatusFromContext(ec))
}

// Trigger sets the HX-Trigger header to the given raw JSON string.
func (b *Builder) Trigger(jsonStr string) *Builder {
	b.headers[htmx.HeaderTrigger] = jsonStr
	return b
}

// TriggerEvent marshals {event: payload} and sets HX-Trigger.
// If marshalling fails, the error is stored and returned by Send().
func (b *Builder) TriggerEvent(event string, payload any) *Builder {
	data := map[string]any{event: payload}
	j, err := json.Marshal(data)
	if err != nil {
		b.err = fmt.Errorf("TriggerEvent: failed to marshal %q: %w", event, err)
		return b
	}
	b.headers[htmx.HeaderTrigger] = string(j)
	return b
}

// Redirect sets HX-Redirect for a client-side HTMX redirect.
func (b *Builder) Redirect(url string) *Builder {
	b.headers[htmx.HeaderRedirect] = url
	return b
}

// PushURL sets HX-Push-Url to push a URL into the browser history.
func (b *Builder) PushURL(url string) *Builder {
	b.headers[htmx.HeaderPushURL] = url
	return b
}

// Reswap sets HX-Reswap to override the swap strategy.
func (b *Builder) Reswap(strategy htmx.SwapStrategy) *Builder {
	b.headers[htmx.HeaderReswap] = string(strategy)
	return b
}

// Retarget sets HX-Retarget to override the target CSS selector.
func (b *Builder) Retarget(selector string) *Builder {
	b.headers[htmx.HeaderRetarget] = selector
	return b
}

// Refresh sets HX-Refresh to trigger a full page reload.
func (b *Builder) Refresh() *Builder {
	b.headers[htmx.HeaderRefresh] = "true"
	return b
}

// Send renders all components into a buffer, then writes headers + body atomically.
// A render failure returns the error without writing anything to the response.
// Also returns any error stored by prior builder methods (e.g. TriggerEvent).
func (b *Builder) Send() error {
	if b.err != nil {
		return b.err
	}
	var buf bytes.Buffer
	ctx := b.c.Request().Context()

	if b.primary != nil {
		if err := b.primary.Render(ctx, &buf); err != nil {
			return err
		}
	}

	for _, oob := range b.oob {
		if err := oob.Render(ctx, &buf); err != nil {
			return err
		}
	}

	resp := b.c.Response()
	for k, v := range b.headers {
		resp.Header().Set(k, v)
	}
	resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp.WriteHeader(b.statusCode)
	_, err := resp.Write(buf.Bytes())
	return err
}
