// Package shared provides common utilities, configuration structures, and shared types.
// It is intended for code and types used across multiple packages.
package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// RequestIDKey is a custom type for context keys to avoid collisions
type RequestIDKey struct{}

// RequestIDKeyValue is the exported value used as a key for storing request IDs in context.Context
var RequestIDKeyValue = RequestIDKey{}

// ContextIDKey is a custom type for context keys to avoid collisions
type ContextIDKey struct{}

// ContextIDKeyValue is the exported value used as a key for storing context IDs in context.Context
var ContextIDKeyValue = ContextIDKey{}

// ContextDescriptionKey is a custom type for context keys to avoid collisions
type ContextDescriptionKey struct{}

// ContextDescriptionKeyValue is the exported value used as a key for storing context descriptions in context.Context
var ContextDescriptionKeyValue = ContextDescriptionKey{}

// RuntimeID is a unique identifier set once at application startup.
// It allows filtering logs for a specific process lifetime (e.g. jq '.runtime_id == "..."').
var RuntimeID string

func init() {
	RuntimeID = GenerateContextID()
}

// GenerateContextID generates a unique context ID for worker execution tracing
func GenerateContextID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

// WithContextID adds a context_id to the context
func WithContextID(ctx context.Context, contextID string) context.Context {
	return context.WithValue(ctx, ContextIDKeyValue, contextID)
}

// WithContextDescription adds a context_description to the context
func WithContextDescription(ctx context.Context, description string) context.Context {
	return context.WithValue(ctx, ContextDescriptionKeyValue, description)
}

// WithContextIDAndDescription adds both context_id and context_description to the context
func WithContextIDAndDescription(ctx context.Context, contextID string, description string) context.Context {
	ctx = WithContextID(ctx, contextID)
	ctx = WithContextDescription(ctx, description)
	return ctx
}
