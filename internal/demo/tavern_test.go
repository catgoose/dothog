// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ReplayLab ---

func TestReplayLab_NextEventIncrements(t *testing.T) {
	rl := NewReplayLab(50)

	id1, seq1 := rl.NextEvent()
	id2, seq2 := rl.NextEvent()

	assert.Equal(t, "replay-1", id1)
	assert.Equal(t, "replay-2", id2)
	assert.Equal(t, int64(1), seq1)
	assert.Equal(t, int64(2), seq2)
}

func TestReplayLab_ResetZeroesCounter(t *testing.T) {
	rl := NewReplayLab(50)
	rl.NextEvent()
	rl.NextEvent()
	rl.Reset()

	id, seq := rl.NextEvent()
	assert.Equal(t, "replay-1", id)
	assert.Equal(t, int64(1), seq)
}

func TestReplayLab_SetReplayWindow(t *testing.T) {
	rl := NewReplayLab(50)
	assert.Equal(t, 50, rl.ReplayWindow())

	rl.SetReplayWindow(10)
	assert.Equal(t, 10, rl.ReplayWindow())
}

// --- BackpressureLab ---

func TestBackpressureLab_DefaultsToCalmPreset(t *testing.T) {
	bl := NewBackpressureLab()
	assert.Equal(t, "calm", bl.ActivePreset())
}

func TestBackpressureLab_SetPresetValid(t *testing.T) {
	bl := NewBackpressureLab()
	bl.SetPreset("heavy")
	assert.Equal(t, "heavy", bl.ActivePreset())
}

func TestBackpressureLab_SetPresetIgnoresUnknown(t *testing.T) {
	bl := NewBackpressureLab()
	bl.SetPreset("not-a-real-preset")
	assert.Equal(t, "calm", bl.ActivePreset(),
		"unknown preset should be ignored")
}

func TestBackpressureLab_RecordTierChangeUpdatesHighestTier(t *testing.T) {
	bl := NewBackpressureLab()
	assert.Equal(t, 0, bl.HighestTier())

	bl.RecordTierChange("topic-a", "sub-1", 0, 1) // throttle
	assert.Equal(t, 1, bl.HighestTier())

	bl.RecordTierChange("topic-b", "sub-2", 0, 2) // simplify
	assert.Equal(t, 2, bl.HighestTier())
}

func TestBackpressureLab_HighestTierDropsOnDisconnect(t *testing.T) {
	bl := NewBackpressureLab()
	bl.RecordTierChange("topic-a", "sub-1", 0, 2) // simplify
	require.Equal(t, 2, bl.HighestTier())

	// Disconnect (tier 3) is treated as eviction — entry is removed.
	bl.RecordTierChange("topic-a", "sub-1", 2, 3)
	assert.Equal(t, 0, bl.HighestTier(),
		"evicted subscriber should not contribute to highest tier")
}

func TestBackpressureLab_TierChangesRingBuffer(t *testing.T) {
	bl := NewBackpressureLab()
	for i := 0; i < 50; i++ {
		bl.RecordTierChange("topic-a", "sub-1", 0, 1)
	}
	changes := bl.TierChanges()
	assert.LessOrEqual(t, len(changes), 30,
		"tier change log should be capped at 30 entries")
}

// --- HooksLab ---

func TestHooksLab_DefaultSource(t *testing.T) {
	hl := NewHooksLab()
	assert.NotEmpty(t, hl.Source())
}

func TestHooksLab_UpdateAndSource(t *testing.T) {
	hl := NewHooksLab()
	hl.Update("new content")
	assert.Equal(t, "new content", hl.Source())
}

func TestHooksLab_RecordHookRingBuffer(t *testing.T) {
	hl := NewHooksLab()
	for i := 0; i < 30; i++ {
		hl.RecordHook("after", "computing")
	}
	log := hl.HookLog()
	assert.LessOrEqual(t, len(log), 20,
		"hook log should be capped at 20 entries")
}

func TestHooksLab_PublishStatsAccumulate(t *testing.T) {
	hl := NewHooksLab()

	count, bytes := hl.PublishStats()
	assert.Equal(t, int64(0), count)
	assert.Equal(t, int64(0), bytes)

	hl.AddPublishStats(100)
	hl.AddPublishStats(50)

	count, bytes = hl.PublishStats()
	assert.Equal(t, int64(2), count)
	assert.Equal(t, int64(150), bytes)
}

// --- PublishStats ---

func TestPublishStats_AddAndSnapshot(t *testing.T) {
	ps := &PublishStats{}

	ps.Add(100)
	ps.Add(50)
	ps.Add(25)

	count, bytes := ps.Snapshot()
	assert.Equal(t, int64(3), count)
	assert.Equal(t, int64(175), bytes)
}

func TestPublishStats_Reset(t *testing.T) {
	ps := &PublishStats{}
	ps.Add(100)
	ps.Add(200)

	ps.Reset()

	count, bytes := ps.Snapshot()
	assert.Equal(t, int64(0), count)
	assert.Equal(t, int64(0), bytes)
}
