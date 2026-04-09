// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionHouse_Items(t *testing.T) {
	h := NewAuctionHouse()
	items := h.Items()
	require.Len(t, items, 8, "expected 8 seeded items")
	// Items should come back sorted by ID.
	for i, item := range items {
		assert.Equal(t, i+1, item.ID)
	}
}

func TestAuctionHouse_Item(t *testing.T) {
	h := NewAuctionHouse()
	item, ok := h.Item(1)
	require.True(t, ok)
	assert.Equal(t, 1, item.ID)
	assert.NotEmpty(t, item.Name)

	_, ok = h.Item(999)
	assert.False(t, ok, "lookup of unknown ID should return false")
}

func TestAuctionHouse_PlaceBid_HappyPath(t *testing.T) {
	h := NewAuctionHouse()
	original, _ := h.Item(1)

	updated, err := h.PlaceBid(1, "Alice", original.CurrentBid+100)
	require.NoError(t, err)
	assert.Equal(t, "Alice", updated.HighBidder)
	assert.Equal(t, original.CurrentBid+100, updated.CurrentBid)
	assert.Equal(t, original.BidCount+1, updated.BidCount)
}

func TestAuctionHouse_PlaceBid_RejectsLowBid(t *testing.T) {
	h := NewAuctionHouse()
	original, _ := h.Item(1)

	_, err := h.PlaceBid(1, "Bob", original.CurrentBid)
	require.Error(t, err, "bid equal to current should fail")

	_, err = h.PlaceBid(1, "Bob", original.CurrentBid-1)
	require.Error(t, err, "bid below current should fail")

	// Item state should be unchanged after a failed bid.
	after, _ := h.Item(1)
	assert.Equal(t, original.CurrentBid, after.CurrentBid)
	assert.Equal(t, original.BidCount, after.BidCount)
	assert.Equal(t, original.HighBidder, after.HighBidder)
}

func TestAuctionHouse_PlaceBid_UnknownItem(t *testing.T) {
	h := NewAuctionHouse()
	_, err := h.PlaceBid(999, "Carol", 100000)
	require.Error(t, err)
}

func TestAuctionHouse_TopicAndAllTopics(t *testing.T) {
	h := NewAuctionHouse()
	assert.Equal(t, "auction/item-1", h.Topic(1))
	assert.Equal(t, "auction/item-8", h.Topic(8))

	all := h.AllTopics()
	require.Len(t, all, 8)
	for i, topic := range all {
		assert.Equal(t, h.Topic(i+1), topic)
	}
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		want  string
		cents int
	}{
		{"$0.00", 0},
		{"$0.01", 1},
		{"$0.99", 99},
		{"$1.00", 100},
		{"$10.50", 1050},
		{"$1000.00", 100000},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, FormatPrice(tc.cents))
	}
}
