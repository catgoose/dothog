// Package shared provides common utilities, configuration structures, and shared types.
// It is intended for code and types used across multiple packages.
package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// RequestIDKey is the context-key type for request IDs (typed to avoid collisions).
type RequestIDKey struct{}

// RequestIDKeyValue is the singleton key value for RequestIDKey.
var RequestIDKeyValue = RequestIDKey{}

// ContextIDKey is the context-key type for context IDs (typed to avoid collisions).
type ContextIDKey struct{}

// ContextIDKeyValue is the singleton key value for ContextIDKey.
var ContextIDKeyValue = ContextIDKey{}

// ContextDescriptionKey is the context-key type for context descriptions (typed to avoid collisions).
type ContextDescriptionKey struct{}

// ContextDescriptionKeyValue is the singleton key value for ContextDescriptionKey.
var ContextDescriptionKeyValue = ContextDescriptionKey{}

// RuntimeID is a unique identifier set once at application startup.
// It allows filtering logs for a specific process lifetime (e.g. jq '.runtime_id == "..."').
var RuntimeID string

func init() {
	RuntimeID = GenerateContextID()
}

// GenerateContextID is a hex-encoded 16-byte random ID; empty string if crypto/rand fails.
func GenerateContextID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

// WithContextID stores contextID on ctx under ContextIDKeyValue so downstream
// loggers can include it in structured fields.
func WithContextID(ctx context.Context, contextID string) context.Context {
	return context.WithValue(ctx, ContextIDKeyValue, contextID)
}

// WithContextDescription stores a human-readable label on ctx under
// ContextDescriptionKeyValue for diagnostic logging.
func WithContextDescription(ctx context.Context, description string) context.Context {
	return context.WithValue(ctx, ContextDescriptionKeyValue, description)
}

// WithContextIDAndDescription is the combined-write helper; later
// retrieval still goes through the individual key lookups.
func WithContextIDAndDescription(ctx context.Context, contextID string, description string) context.Context {
	ctx = WithContextID(ctx, contextID)
	ctx = WithContextDescription(ctx, description)
	return ctx
}
