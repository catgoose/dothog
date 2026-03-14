package requestlog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"catgoose/dothog/internal/shared"
)

// Handler is a slog.Handler that captures log records into a Store
// when the record is associated with a request ID (via WithAttrs or context).
type Handler struct {
	inner     slog.Handler
	store     *Store
	requestID string // set by WithAttrs when "request_id" is added
	attrs     []slog.Attr
}

// NewHandler wraps an existing slog.Handler so that every record with a
// request_id attribute is also stored in the given Store.
func NewHandler(inner slog.Handler, store *Store) *Handler {
	return &Handler{inner: inner, store: store}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Determine request ID from handler attrs or context.
	reqID := h.requestID
	if reqID == "" {
		if id, ok := ctx.Value(shared.RequestIDKeyValue).(string); ok {
			reqID = id
		}
	}

	if reqID != "" {
		// Collect extra attrs from the record itself.
		var parts []string
		for _, a := range h.attrs {
			if a.Key != "request_id" {
				parts = append(parts, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
			}
		}
		r.Attrs(func(a slog.Attr) bool {
			if a.Key != "request_id" {
				parts = append(parts, fmt.Sprintf("%s=%s", a.Key, a.Value.String()))
			}
			return true
		})

		h.store.Append(reqID, Entry{
			Time:    r.Time,
			Level:   r.Level.String(),
			Message: r.Message,
			Attrs:   strings.Join(parts, " "),
		})
	}

	return h.inner.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	reqID := h.requestID
	for _, a := range attrs {
		if a.Key == "request_id" {
			reqID = a.Value.String()
			break
		}
	}
	return &Handler{
		inner:     h.inner.WithAttrs(attrs),
		store:     h.store,
		requestID: reqID,
		attrs:     append(cloneAttrs(h.attrs), attrs...),
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		inner:     h.inner.WithGroup(name),
		store:     h.store,
		requestID: h.requestID,
		attrs:     cloneAttrs(h.attrs),
	}
}

func cloneAttrs(src []slog.Attr) []slog.Attr {
	if len(src) == 0 {
		return nil
	}
	dst := make([]slog.Attr, len(src))
	copy(dst, src)
	return dst
}
