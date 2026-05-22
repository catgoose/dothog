// setup:feature:demo

package demo

import (
	"fmt"
	"sync"
	"time"
)

// AuctionItem holds a single lot's bid state; prices are stored in cents to avoid float drift.
type AuctionItem struct {
	EndsAt      time.Time
	Name        string
	Description string
	ImageEmoji  string
	HighBidder  string
	ID          int
	StartPrice  int
	CurrentBid  int
	BidCount    int
}

// AuctionHouse is a thread-safe in-memory catalog of AuctionItems indexed by ID.
type AuctionHouse struct {
	items map[int]*AuctionItem
	mu    sync.RWMutex
}

// FormatPrice renders cents as "$D.CC"; integer math avoids float drift.
func FormatPrice(cents int) string {
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

// NewAuctionHouse pre-seeds 8 items with staggered end times across the next few hours.
func NewAuctionHouse() *AuctionHouse {
	now := time.Now()
	items := []*AuctionItem{
		{ID: 1, Name: "Vintage Camera", Description: "A pristine 1960s rangefinder camera", ImageEmoji: "\U0001F4F7", StartPrice: 5000, CurrentBid: 5000, EndsAt: now.Add(2 * time.Hour)},
		{ID: 2, Name: "Signed First Edition", Description: "A rare signed copy of a literary classic", ImageEmoji: "\U0001F4DA", StartPrice: 12000, CurrentBid: 12000, EndsAt: now.Add(3 * time.Hour)},
		{ID: 3, Name: "Rare Gold Coin", Description: "An 1849 gold dollar in mint condition", ImageEmoji: "\U0001FA99", StartPrice: 20000, CurrentBid: 20000, EndsAt: now.Add(4 * time.Hour)},
		{ID: 4, Name: "Mechanical Watch", Description: "Swiss-made automatic movement, circa 1972", ImageEmoji: "\u231A", StartPrice: 8500, CurrentBid: 8500, EndsAt: now.Add(2*time.Hour + 30*time.Minute)},
		{ID: 5, Name: "Antique Map", Description: "Hand-drawn map of the Mediterranean, 1780", ImageEmoji: "\U0001F5FA", StartPrice: 6500, CurrentBid: 6500, EndsAt: now.Add(5 * time.Hour)},
		{ID: 6, Name: "Vinyl Record Collection", Description: "50 pristine jazz and blues LPs from the 1950s", ImageEmoji: "\U0001F3B5", StartPrice: 4000, CurrentBid: 4000, EndsAt: now.Add(1*time.Hour + 45*time.Minute)},
		{ID: 7, Name: "Handmade Ceramic Vase", Description: "Glazed stoneware by a master potter", ImageEmoji: "\U0001F3FA", StartPrice: 3000, CurrentBid: 3000, EndsAt: now.Add(6 * time.Hour)},
		{ID: 8, Name: "Retro Game Console", Description: "Fully working console with 12 original cartridges", ImageEmoji: "\U0001F3AE", StartPrice: 7500, CurrentBid: 7500, EndsAt: now.Add(3*time.Hour + 15*time.Minute)},
	}
	m := make(map[int]*AuctionItem, len(items))
	for _, item := range items {
		m[item.ID] = item
	}
	return &AuctionHouse{items: m}
}

// Items is a value-copy snapshot sorted by ID; safe to mutate independently of the house.
func (h *AuctionHouse) Items() []AuctionItem {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]AuctionItem, 0, len(h.items))
	for i := 1; i <= len(h.items); i++ {
		if item, ok := h.items[i]; ok {
			out = append(out, *item)
		}
	}
	return out
}

// Item returns a copy; mutations don't reach the house.
func (h *AuctionHouse) Item(id int) (AuctionItem, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	item, ok := h.items[id]
	if !ok {
		return AuctionItem{}, false
	}
	return *item, true
}

// PlaceBid validates and places a bid on an auction item.
func (h *AuctionHouse) PlaceBid(id int, bidder string, amount int) (AuctionItem, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	item, ok := h.items[id]
	if !ok {
		return AuctionItem{}, fmt.Errorf("item %d not found", id)
	}
	if amount <= item.CurrentBid {
		return AuctionItem{}, fmt.Errorf("bid must exceed current price of %s", FormatPrice(item.CurrentBid))
	}
	item.CurrentBid = amount
	item.BidCount++
	item.HighBidder = bidder
	return *item, nil
}

// Topic is the SSE channel name for the given auction item ID ("auction/item-N").
func (h *AuctionHouse) Topic(id int) string {
	return fmt.Sprintf("auction/item-%d", id)
}

// AllTopics is the per-item SSE topic list in ID order, for pre-seeding subscribers.
func (h *AuctionHouse) AllTopics() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.items))
	for i := 1; i <= len(h.items); i++ {
		out = append(out, fmt.Sprintf("auction/item-%d", i))
	}
	return out
}
