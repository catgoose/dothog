// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
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
	broker  *tavern.SSEBroker
	tracker *presence.Tracker
	filters *demo.NotificationFilters
	counter atomic.Int64
}

func (ar *appRoutes) initNotificationsRoutes(broker *tavern.SSEBroker) {
	filters := demo.NewNotificationFilters()

	n := &notificationRoutes{
		broker:  broker,
		filters: filters,
	}

	n.tracker = presence.New(broker, presence.Config{
		StaleTimeout:        30 * time.Second,
		PresenceTopicSuffix: ":presence",
		TargetID:            "presence-list",
		RenderFunc: func(topic string, users []presence.Info) string {
			vu := make([]views.NotifPresenceUser, len(users))
			for i, u := range users {
				vu[i] = views.NotifPresenceUser{
					ID:    u.UserID,
					Name:  u.Name,
					Color: u.Avatar,
				}
			}
			// We render without a "current user" context here — the presence
			// list is broadcast to all subscribers so "(you)" is omitted in
			// the OOB swap. The initial render in the page template shows it.
			return renderPresenceHTML(vu, "")
		},
	})

	ar.e.GET("/realtime/notifications", n.handlePage)
	ar.e.GET("/sse/notifications", n.handleSSE)
	ar.e.POST("/realtime/notifications/filter", n.handleFilterUpdate)

	broker.RunPublisher(ar.ctx, n.startSimulator)
}

func (n *notificationRoutes) handlePage(c echo.Context) error {
	identity := getOrCreateNotifIdentity(c)
	filters := n.filters.EnabledCategories(identity.ID)
	return handler.RenderBaseLayout(c, views.NotificationsPage(identity, filters))
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

	// Join presence
	n.tracker.Join(TopicNotifications, presence.Info{
		UserID: identity.ID,
		Name:   identity.Name,
		Avatar: identity.Color,
	})
	defer n.tracker.Leave(TopicNotifications, identity.ID)

	// Each user gets a dedicated topic so that replay via Last-Event-ID is
	// inherently scoped — no risk of leaking another user's notifications.
	userTopic := notifUserTopic(identity.ID)
	n.broker.SetReplayPolicy(userTopic, 20)

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

	// Check for Last-Event-ID for replay
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
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		case <-heartbeat.C:
			n.tracker.Heartbeat(TopicNotifications, identity.ID)
			_, _ = fmt.Fprintf(c.Response(), ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (n *notificationRoutes) handleFilterUpdate(c echo.Context) error {
	identity := getOrCreateNotifIdentity(c)
	cat := demo.NotificationCategory(c.FormValue("category"))
	// Toggle: if "disabled" is set, turn off; otherwise turn on
	enabled := c.FormValue("disabled") == ""
	n.filters.SetFilter(identity.ID, cat, enabled)
	return c.NoContent(http.StatusNoContent)
}

func (n *notificationRoutes) startSimulator(ctx context.Context) {
	for {
		delay := time.Duration(2000+rand.IntN(3000)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			online := n.tracker.List(TopicNotifications)
			if len(online) == 0 {
				continue
			}
			target := online[rand.IntN(len(online))]

			cat := demo.AllNotificationCategories[rand.IntN(len(demo.AllNotificationCategories))]
			message := demo.FormatNotification(cat)
			notifID := fmt.Sprintf("n%d", n.counter.Add(1))
			timestamp := time.Now().Format("15:04:05")

			html := renderNotifItemHTML(notifID, cat, message, timestamp)

			sseMsg := tavern.NewSSEMessage("notification", html).
				WithID(notifID).
				String()

			n.broker.PublishWithTTL(
				notifUserTopic(target.UserID),
				sseMsg,
				60*time.Second,
				tavern.WithAutoRemove("notif-"+notifID),
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
