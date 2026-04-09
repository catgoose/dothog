// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	appenv "catgoose/dothog/internal/env"
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/catgoose/tavern/presence"
	"github.com/labstack/echo/v4"
)

const notifCookie = "notif_identity"

type notificationRoutes struct {
	broker   *tavern.SSEBroker
	tracker  *presence.Tracker
	filters  *demo.NotificationFilters
	counter  atomic.Int64
	paused   atomic.Bool
	minDelay atomic.Int64 // nanoseconds
	maxDelay atomic.Int64 // nanoseconds
}

func (ar *appRoutes) initNotificationsRoutes(broker *tavern.SSEBroker) {
	filters := demo.NewNotificationFilters()

	n := &notificationRoutes{
		broker:  broker,
		filters: filters,
	}
	n.minDelay.Store(int64(2 * time.Second))
	n.maxDelay.Store(int64(5 * time.Second))

	n.tracker = presence.New(broker, presence.Config{
		StaleTimeout:        30 * time.Second,
		PresenceTopicSuffix: ":presence",
		TargetID:            "presence-list",
		RenderFunc: func(topic string, users []presence.Info) string {
			// Multiple SSE connections may share the same identity (e.g.
			// multiple tabs). Collapse by identity ID so each user appears
			// once in the rendered list. Sort by identity ID for stable
			// rendering order across reconnects.
			seen := make(map[string]struct{})
			var vu []views.NotifPresenceUser
			for _, u := range users {
				identityID, _ := u.Metadata["identity_id"].(string)
				if identityID == "" {
					identityID = u.UserID
				}
				if _, ok := seen[identityID]; ok {
					continue
				}
				seen[identityID] = struct{}{}
				vu = append(vu, views.NotifPresenceUser{
					ID:    identityID,
					Name:  u.Name,
					Color: u.Avatar,
				})
			}
			sort.Slice(vu, func(i, j int) bool {
				return vu[i].ID < vu[j].ID
			})
			// We render without a "current user" context here — the presence
			// list is broadcast to all subscribers so "(you)" is omitted in
			// the OOB swap. The initial render in the page template shows it.
			return renderPresenceHTML(vu, "")
		},
	})

	ar.e.GET("/realtime/notifications", n.handlePage)
	ar.e.GET("/sse/notifications", n.handleSSE)
	ar.e.POST("/realtime/notifications/filter", n.handleFilterUpdate)
	ar.e.POST("/realtime/notifications/identity", n.handleIdentitySwitch)
	ar.e.POST("/realtime/notifications/simulator/pause", n.handleSimulatorPause)
	ar.e.POST("/realtime/notifications/simulator/speed", n.handleSimulatorSpeed)

	broker.RunPublisher(ar.ctx, n.startSimulator)
}

func (n *notificationRoutes) handlePage(c echo.Context) error {
	identity := getOrCreateNotifIdentity(c)
	filters := n.filters.EnabledCategories(identity.ID)
	state := views.NotifSimulatorState{
		Paused:   n.paused.Load(),
		MinDelay: time.Duration(n.minDelay.Load()),
		MaxDelay: time.Duration(n.maxDelay.Load()),
	}
	return handler.RenderBaseLayout(c, views.NotificationsPage(identity, filters, state))
}

func (n *notificationRoutes) handleIdentitySwitch(c echo.Context) error {
	id := c.FormValue("identity")
	idx := demo.IdentityIndexByID(id)
	if idx < 0 {
		return c.String(http.StatusBadRequest, "unknown identity")
	}
	c.SetCookie(&http.Cookie{
		Name:     notifCookie,
		Value:    fmt.Sprintf("%d", idx),
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
	identity := demo.AssignIdentity(idx)
	filters := n.filters.EnabledCategories(identity.ID)
	state := views.NotifSimulatorState{
		Paused:   n.paused.Load(),
		MinDelay: time.Duration(n.minDelay.Load()),
		MaxDelay: time.Duration(n.maxDelay.Load()),
	}
	return handler.RenderComponent(c, views.NotificationsPage(identity, filters, state))
}

func (n *notificationRoutes) handleSimulatorPause(c echo.Context) error {
	n.paused.Store(!n.paused.Load())
	if n.paused.Load() {
		return c.HTML(http.StatusOK, "Resume")
	}
	return c.HTML(http.StatusOK, "Pause")
}

func (n *notificationRoutes) handleSimulatorSpeed(c echo.Context) error {
	var minMs, maxMs int
	switch c.FormValue("speed") {
	case "slow":
		minMs, maxMs = 5000, 10000
	case "normal":
		minMs, maxMs = 2000, 5000
	case "fast":
		minMs, maxMs = 500, 1500
	case "flood":
		minMs, maxMs = 50, 200
	default:
		return c.String(http.StatusBadRequest, "unknown speed")
	}
	n.minDelay.Store(int64(time.Duration(minMs) * time.Millisecond))
	n.maxDelay.Store(int64(time.Duration(maxMs) * time.Millisecond))
	return c.NoContent(http.StatusNoContent)
}

func (n *notificationRoutes) handleSSE(c echo.Context) error {
	identity := getOrCreateNotifIdentity(c)

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	// Each SSE connection gets a unique presence key so that multiple tabs
	// sharing the same identity cookie track independently. The real identity
	// ID is stored in metadata for notification routing and deduplication.
	b := make([]byte, 4)
	_, _ = crand.Read(b)
	connID := hex.EncodeToString(b)

	n.tracker.Join(TopicNotifications, presence.Info{
		UserID: connID,
		Name:   identity.Name,
		Avatar: identity.Color,
		Metadata: map[string]any{
			"identity_id": identity.ID,
		},
	})
	defer n.tracker.Leave(TopicNotifications, connID)

	// Each user gets a dedicated topic so that replay via Last-Event-ID is
	// inherently scoped — no risk of leaking another user's notifications.
	userTopic := notifUserTopic(identity.ID)
	n.broker.SetReplayPolicy(userTopic, 50)
	n.broker.SetReplayGapPolicy(userTopic, tavern.GapFallbackToSnapshot, nil)

	// Build a filter that checks the user's current category preferences.
	// The rendered HTML includes data-cat="<category>" so we can detect it.
	filterFn := func(msg string) bool {
		for _, cat := range demo.AllNotificationCategories {
			if strings.Contains(msg, fmt.Sprintf(`data-cat="%s"`, string(cat))) {
				return n.filters.IsEnabled(identity.ID, cat)
			}
		}
		return true
	}

	// Check for Last-Event-ID for replay.
	// Both paths apply the same filterFn: SubscribeWith uses it natively for
	// live messages, while SubscribeFromID doesn't support filters so we apply
	// the filter in the write loop below. This ensures category preferences
	// survive SSE reconnections.
	lastEventID := c.Request().Header.Get("Last-Event-ID")
	var msgs <-chan string
	var unsub func()
	if lastEventID != "" {
		msgs, unsub = n.broker.SubscribeFromID(userTopic, lastEventID)
	} else {
		msgs, unsub = n.broker.SubscribeWith(userTopic,
			tavern.SubWithFilter(filterFn),
		)
	}
	defer unsub()

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			if !filterFn(msg) {
				continue
			}
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		case <-heartbeat.C:
			n.tracker.Heartbeat(TopicNotifications, connID)
			_, _ = fmt.Fprintf(c.Response(), ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (n *notificationRoutes) handleFilterUpdate(c echo.Context) error {
	identity := getOrCreateNotifIdentity(c)
	cat := demo.NotificationCategory(c.FormValue("category"))
	enabled := c.FormValue("enabled") == "true"
	n.filters.SetFilter(identity.ID, cat, enabled)
	return c.NoContent(http.StatusNoContent)
}

func (n *notificationRoutes) startSimulator(ctx context.Context) {
	for {
		minD := time.Duration(n.minDelay.Load())
		maxD := time.Duration(n.maxDelay.Load())
		if maxD < minD {
			maxD = minD
		}
		spread := maxD - minD
		var delay time.Duration
		if spread > 0 {
			delay = minD + time.Duration(rand.Int64N(int64(spread)))
		} else {
			delay = minD
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if n.paused.Load() {
				continue
			}
			online := n.tracker.List(TopicNotifications)
			if len(online) == 0 {
				continue
			}
			target := online[rand.IntN(len(online))]

			// The presence UserID is a per-connection key; the real identity
			// ID used for notification routing lives in metadata.
			identityID, _ := target.Metadata["identity_id"].(string)
			if identityID == "" {
				continue
			}

			cat := demo.AllNotificationCategories[rand.IntN(len(demo.AllNotificationCategories))]
			message := demo.FormatNotification(cat)
			notifID := fmt.Sprintf("n%d", n.counter.Add(1))
			timestamp := time.Now().Format("15:04:05")

			html := renderNotifItemHTML(notifID, cat, message, timestamp)

			sseMsg := tavern.NewSSEMessage("notification", html).
				WithID(notifID).
				String()

			n.broker.PublishWithID(
				notifUserTopic(identityID),
				notifID,
				sseMsg,
			)
		}
	}
}

func renderNotifItemHTML(id string, cat demo.NotificationCategory, message, timestamp string) string {
	buf := &bytes.Buffer{}
	cmp := views.NotificationItem(id, cat, message, timestamp)
	if err := cmp.Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render notification"), buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderPresenceHTML(users []views.NotifPresenceUser, currentUserID string) string {
	buf := &bytes.Buffer{}
	cmp := views.PresenceList(users, currentUserID)
	if err := cmp.Render(shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render presence"), buf); err != nil {
		return ""
	}
	return buf.String()
}

// notifUserTopic returns a per-user notification topic so that publish, subscribe,
// and Last-Event-ID replay are all inherently scoped to a single user.
func notifUserTopic(userID string) string {
	return TopicNotifications + "-" + userID
}

func getOrCreateNotifIdentity(c echo.Context) demo.NotificationIdentity {
	if cookie, err := c.Cookie(notifCookie); err == nil && cookie.Value != "" {
		// Parse index from cookie value
		var idx int
		if _, err := fmt.Sscanf(cookie.Value, "%d", &idx); err == nil {
			return demo.AssignIdentity(idx)
		}
	}
	idx := demo.RandomIdentityIndex()
	c.SetCookie(&http.Cookie{
		Name:     notifCookie,
		Value:    fmt.Sprintf("%d", idx),
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
	return demo.AssignIdentity(idx)
}
