// setup:feature:demo

package routes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLooksLikeFailuresEventID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"evt-1", true},
		{"evt-42", true},
		{"evt-99999", true},
		{"evt-0", true},
		{"evt-", false},     // no digits
		{"evt", false},      // too short
		{"", false},         // empty
		{"EVT-1", false},    // wrong case
		{"evt-abc", false},  // non-numeric suffix
		{"evt-1x", false},   // trailing non-digit
		{"this-is-not-a-valid-id", false},
		{"foo-123", false},  // wrong prefix
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.want, looksLikeFailuresEventID(tt.id))
		})
	}
}

func TestResumeDescriptionFrame_Malformed(t *testing.T) {
	frame := resumeDescriptionFrame("this-is-not-a-valid-id")

	// Should contain the malformed classification.
	assert.Contains(t, frame, "Malformed Last-Event-ID")
	// Should include the actual token in the detail.
	assert.Contains(t, frame, "this-is-not-a-valid-id")
	// Should be an SSE event targeting failures-result.
	assert.Contains(t, frame, "event: failures-result")
	// Should describe broker behavior.
	assert.Contains(t, frame, "gap policy")
}

func TestResumeDescriptionFrame_ValidShaped(t *testing.T) {
	frame := resumeDescriptionFrame("evt-42")

	// Should classify as a normal resume attempt.
	assert.Contains(t, frame, "Resume attempted")
	assert.Contains(t, frame, "evt-42")
	assert.Contains(t, frame, "event: failures-result")
	// Should NOT say malformed.
	assert.NotContains(t, frame, "Malformed")
	// Should describe the broker lookup path.
	assert.Contains(t, frame, "replay buffer")
}

func TestResumeDescriptionFrame_SSEFormat(t *testing.T) {
	frame := resumeDescriptionFrame("evt-1")

	// SSE messages must have event: and data: lines.
	require.True(t, strings.Contains(frame, "event:"))
	require.True(t, strings.Contains(frame, "data:"))
}

func TestFailuresRoutes_GetLatestID_Empty(t *testing.T) {
	r := &failuresRoutes{}
	assert.Equal(t, "", r.getLatestID())
}

func TestFailuresRoutes_GetLatestID_AfterStore(t *testing.T) {
	r := &failuresRoutes{}
	r.latestID.Store("evt-5")
	assert.Equal(t, "evt-5", r.getLatestID())
}
