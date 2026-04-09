// setup:feature:demo

package demo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservatoryState_DefaultMaxPerTopic(t *testing.T) {
	s := NewObservatoryState()
	assert.Equal(t, 10, s.MaxPerTopic())
}

func TestObservatoryState_SetMaxPerTopic(t *testing.T) {
	s := NewObservatoryState()
	s.SetMaxPerTopic(50)
	assert.Equal(t, 50, s.MaxPerTopic())

	s.SetMaxPerTopic(0)
	assert.Equal(t, 0, s.MaxPerTopic())

	// Negative values are clamped to zero.
	s.SetMaxPerTopic(-5)
	assert.Equal(t, 0, s.MaxPerTopic())
}

func TestObservatoryState_StressLifecycle(t *testing.T) {
	s := NewObservatoryState()
	assert.False(t, s.StressActive())

	_, cancel := context.WithCancel(context.Background())
	s.SetStress(true, cancel)
	assert.True(t, s.StressActive())

	s.CancelStress()
	assert.False(t, s.StressActive())
}

func TestObservatoryState_RecordTierChangeRingBuffer(t *testing.T) {
	s := NewObservatoryState()
	for i := 0; i < maxTierChanges*2; i++ {
		s.RecordTierChange("topic-a", "sub-1", 1)
	}
	changes := s.RecentTierChanges()
	assert.LessOrEqual(t, len(changes), maxTierChanges,
		"tier change log should be capped at maxTierChanges")
}

func TestObservatoryState_RecentTierChangesNewestFirst(t *testing.T) {
	s := NewObservatoryState()
	s.RecordTierChange("topic-a", "sub-1", 1)
	s.RecordTierChange("topic-b", "sub-2", 2)
	s.RecordTierChange("topic-c", "sub-3", 3)

	changes := s.RecentTierChanges()
	require.Len(t, changes, 3)
	// Newest first.
	assert.Equal(t, "topic-c", changes[0].Topic)
	assert.Equal(t, "topic-b", changes[1].Topic)
	assert.Equal(t, "topic-a", changes[2].Topic)
}

func TestTierName(t *testing.T) {
	assert.Equal(t, "Normal", TierName(0))
	assert.Equal(t, "Throttle", TierName(1))
	assert.Equal(t, "Simplify", TierName(2))
	assert.Equal(t, "Disconnect", TierName(3))
	assert.Equal(t, "Unknown", TierName(99))
}
