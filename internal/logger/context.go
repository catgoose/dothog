package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// contextIDKey and contextDescriptionKey are the singleton context-key values
// the structured logger pulls from each request's ctx in WithContext, so
// background jobs and ad-hoc render contexts can carry their own diagnostic
// identity alongside per-request log lines.
type (
	contextIDKey          struct{}
	contextDescriptionKey struct{}
)

// ContextIDKeyValue and ContextDescriptionKeyValue are the keys WithContext
// reads. Callers that need to stamp a value into ctx directly (without the
// With* helpers) reference these.
var (
	ContextIDKeyValue          = contextIDKey{}
	ContextDescriptionKeyValue = contextDescriptionKey{}
)

// GenerateContextID returns a hex-encoded 16-byte random ID; empty string if
// crypto/rand fails. Suitable for per-call diagnostic IDs surfaced through
// WithContext as context_id.
func GenerateContextID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

// WithContextID stores contextID on ctx so downstream WithContext loggers
// include it in structured fields as context_id.
func WithContextID(ctx context.Context, contextID string) context.Context {
	return context.WithValue(ctx, ContextIDKeyValue, contextID)
}

// WithContextDescription stores a human-readable label on ctx surfaced through
// WithContext as context_description.
func WithContextDescription(ctx context.Context, description string) context.Context {
	return context.WithValue(ctx, ContextDescriptionKeyValue, description)
}

// WithContextIDAndDescription stamps both diagnostic fields in one call;
// retrieval still goes through the individual key lookups.
func WithContextIDAndDescription(ctx context.Context, contextID string, description string) context.Context {
	ctx = WithContextID(ctx, contextID)
	ctx = WithContextDescription(ctx, description)
	return ctx
}
