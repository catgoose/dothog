// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	appenv "catgoose/dothog/internal/env"
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

var auctionCounter atomic.Int64

const auctionWatchCookie = "auction_watched"

var botNames = []string{
	"AutoBid-7", "BidBot-3", "Sniper-9", "DealHawk-2",
	"AuctionAce-5", "BargainBot-1", "FlipMaster-8", "QuickBid-4",
}

type auctionRoutes struct {
	broker *tavern.SSEBroker
	house  *demo.AuctionHouse
}

func (ar *appRoutes) initAuctionRoutes(broker *tavern.SSEBroker) {
	house := demo.NewAuctionHouse()
	a := &auctionRoutes{broker: broker, house: house}

	// Set replay policy on each auction topic so reconnecting clients
	// receive the last 5 bids.
	for _, topic := range house.AllTopics() {
		broker.SetReplayPolicy(topic, 5)
	}

	// Dynamic group: per-request topic resolution based on watched items cookie.
	broker.DynamicGroup("my-auctions", func(r *http.Request) []string {
		return a.watchedTopicsFromRequest(r)
	})

	ar.e.GET("/realtime/auction", a.handlePage)
	ar.e.GET("/sse/auction", echo.WrapHandler(broker.DynamicGroupHandler("my-auctions")))
	ar.e.POST("/realtime/auction/bid", a.handleBid)
	ar.e.POST("/realtime/auction/watch", a.handleWatchToggle)

	broker.RunPublisher(ar.ctx, a.startBotBidder)
}

func (a *auctionRoutes) handlePage(c echo.Context) error {
	watched := a.watchedSet(c)
	items := a.house.Items()
	return handler.RenderBaseLayout(c, views.AuctionPage(items, watched))
}

func (a *auctionRoutes) handleBid(c echo.Context) error {
	itemID, err := strconv.Atoi(c.FormValue("item_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid item ID")
	}
	bidder := strings.TrimSpace(c.FormValue("bidder"))
	if bidder == "" {
		bidder = "Anonymous"
	}
	amountStr := c.FormValue("amount")
	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid bid amount")
	}
	amountCents := int(amountFloat * 100)

	item, err := a.house.PlaceBid(itemID, bidder, amountCents)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	topic := a.house.Topic(itemID)
	html := renderAuctionCardUpdateHTML(item)
	eventID := fmt.Sprintf("ab%d", auctionCounter.Add(1))
	a.broker.PublishWithID(topic, eventID, html)

	return c.HTML(http.StatusOK, html)
}

func (a *auctionRoutes) handleWatchToggle(c echo.Context) error {
	itemID, err := strconv.Atoi(c.FormValue("item_id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid item ID")
	}

	watched := a.watchedSet(c)
	_, isWatched := watched[itemID]
	if isWatched {
		delete(watched, itemID)
	} else {
		watched[itemID] = true
	}

	// Persist updated watch list to cookie.
	a.setWatchCookie(c, watched)

	// Return the updated watch button so the client can swap it.
	item, ok := a.house.Item(itemID)
	if !ok {
		return c.String(http.StatusNotFound, "item not found")
	}
	nowWatched := !isWatched
	buf := &bytes.Buffer{}
	cmp := views.AuctionWatchButton(item, nowWatched)
	if err := cmp.Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render watch button"), buf); err != nil {
		return c.String(http.StatusInternalServerError, "render error")
	}
	return c.HTML(http.StatusOK, buf.String())
}

func (a *auctionRoutes) startBotBidder(ctx context.Context) {
	for {
		delay := time.Duration(3000+rand.IntN(5000)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			items := a.house.Items()
			if len(items) == 0 {
				continue
			}
			target := items[rand.IntN(len(items))]
			topic := a.house.Topic(target.ID)

			// Only bid if someone is watching.
			if !a.broker.HasSubscribers(topic) {
				continue
			}

			// Bid 5-15% above current price.
			pct := 5 + rand.IntN(11)
			bump := target.CurrentBid * pct / 100
			if bump < 50 {
				bump = 50 // minimum 50 cents
			}
			newBid := target.CurrentBid + bump
			botName := botNames[rand.IntN(len(botNames))]

			item, err := a.house.PlaceBid(target.ID, botName, newBid)
			if err != nil {
				continue
			}

			html := renderAuctionCardUpdateHTML(item)
			eventID := fmt.Sprintf("ab%d", auctionCounter.Add(1))
			a.broker.PublishWithID(topic, eventID, html)
		}
	}
}

// watchedTopicsFromRequest reads the watched cookie from an http.Request
// and returns the corresponding SSE topics.
func (a *auctionRoutes) watchedTopicsFromRequest(r *http.Request) []string {
	cookie, err := r.Cookie(auctionWatchCookie)
	if err != nil || cookie.Value == "" {
		return nil
	}
	var topics []string
	for _, s := range strings.Split(cookie.Value, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		topics = append(topics, a.house.Topic(id))
	}
	return topics
}

// watchedSet returns the set of watched item IDs from the cookie.
func (a *auctionRoutes) watchedSet(c echo.Context) map[int]bool {
	cookie, err := c.Cookie(auctionWatchCookie)
	if err != nil || cookie.Value == "" {
		return make(map[int]bool)
	}
	watched := make(map[int]bool)
	for _, s := range strings.Split(cookie.Value, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			continue
		}
		watched[id] = true
	}
	return watched
}

// setWatchCookie persists the watched item IDs as a comma-separated cookie.
func (a *auctionRoutes) setWatchCookie(c echo.Context, watched map[int]bool) {
	var ids []string
	for id := range watched {
		ids = append(ids, strconv.Itoa(id))
	}
	c.SetCookie(&http.Cookie{
		Name:     auctionWatchCookie,
		Value:    strings.Join(ids, ","),
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: false, // JS needs to read this for SSE reconnect
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
}

func renderAuctionCardUpdateHTML(item demo.AuctionItem) string {
	buf := &bytes.Buffer{}
	cmp := views.AuctionCardUpdate(item)
	if err := cmp.Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render auction card"), buf); err != nil {
		return ""
	}
	return buf.String()
}
