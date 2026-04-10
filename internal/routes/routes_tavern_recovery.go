// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// Topics for the recovery lab. Each topic exercises a different recovery
// strategy on a single SSE connection.
const (
	topicRecoveryReplay   = "recovery/replay"
	topicRecoverySnapshot = "recovery/snapshot"
	topicRecoveryLive     = "recovery/live"
)

type recoveryRoutes struct {
	broker *tavern.SSEBroker
	lab    *demo.RecoveryLab
}

func (ar *appRoutes) initRecoveryRoutes(broker *tavern.SSEBroker) {
	r := &recoveryRoutes{
		broker: broker,
		lab:    demo.NewRecoveryLab(),
	}

	// Replay topic gets a small replay buffer so reconnecting clients can
	// recover via Last-Event-ID. Snapshot topic doesn't use replay; it relies
	// on per-subscriber snapshot delivery. Live topic gets neither.
	broker.SetReplayPolicy(topicRecoveryReplay, 50)

	ar.e.GET("/realtime/tavern/recovery", r.handlePage)
	ar.e.GET("/sse/tavern/recovery", r.handleSSE)
	ar.e.POST("/realtime/tavern/recovery/snapshot", r.handleSetSnapshot)
	ar.e.POST("/realtime/tavern/recovery/reset", r.handleReset)

	broker.RunPublisher(ar.ctx, r.startPublisher)
}

func (r *recoveryRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.TavernRecoveryPage(r.lab.Snapshot()))
}

func (r *recoveryRoutes) handleSetSnapshot(c echo.Context) error {
	value := strings.TrimSpace(c.FormValue("value"))
	if value == "" {
		return c.String(http.StatusBadRequest, "value is required")
	}
	r.lab.SetSnapshot(value)
	// Publish the new snapshot so currently-connected clients see the change
	// in their snapshot region (in addition to fresh subscribers receiving it
	// on connect via SubscribeWithSnapshot).
	html := renderRecoverySnapshotEvent(value, time.Now().Format("15:04:05.000"), false)
	r.broker.Publish(topicRecoverySnapshot, tavern.NewSSEMessage("recovery-snapshot", html).String())
	return c.NoContent(http.StatusNoContent)
}

func (r *recoveryRoutes) handleReset(c echo.Context) error {
	r.lab.Reset()
	r.broker.ClearReplay(topicRecoveryReplay)
	return c.NoContent(http.StatusNoContent)
}

// handleSSE multiplexes all three recovery topics into one SSE response.
// On reconnect, the replay region uses Last-Event-ID, the snapshot region
// gets the current snapshot value, and the live region just resumes streaming.
//
// StreamSSE takes a single channel, so the three topic channels are fanned
// in to one stream channel. The fan-in goroutine exits when ctx is cancelled
// or any source channel is closed; closing fanIn signals StreamSSE to return.
func (r *recoveryRoutes) handleSSE(c echo.Context) error {
	lastEventID := c.Request().Header.Get("Last-Event-ID")
	reconnect := lastEventID != ""

	// Replay subscription: SubscribeFromIDWith handles both fresh and
	// resumed connections in a single call. The snapshot and live legs
	// below intentionally use different APIs because they exercise
	// different recovery strategies on this multi-region demo.
	replayCh, replayUnsub := r.broker.SubscribeFromIDWith(topicRecoveryReplay, lastEventID)
	defer replayUnsub()

	// Snapshot subscription: deliver the current snapshot atomically before
	// any live events. The snapshot HTML is tagged so the client can show
	// "SNAPSHOT" rather than "LIVE" on the first delivery.
	snapCh, snapUnsub := r.broker.SubscribeWithSnapshot(topicRecoverySnapshot, func() string {
		html := renderRecoverySnapshotEvent(r.lab.Snapshot(), time.Now().Format("15:04:05.000"), true)
		return tavern.NewSSEMessage("recovery-snapshot", html).String()
	})
	defer snapUnsub()

	// Live subscription: no replay, no snapshot.
	liveCh, liveUnsub := r.broker.Subscribe(topicRecoveryLive)
	defer liveUnsub()

	ctx := c.Request().Context()
	fanIn := make(chan string, 16)
	go func() {
		defer close(fanIn)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-replayCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			case msg, ok := <-snapCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			case msg, ok := <-liveCh:
				if !ok {
					return
				}
				select {
				case fanIn <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Tell the client whether this was a fresh connect or a reconnect, so
	// the regions can show the right "first delivery" badges. Delivered as
	// a one-shot snapshot before any channel values are streamed.
	mode := "fresh"
	if reconnect {
		mode = "reconnect"
	}
	modeFrame := fmt.Sprintf("event: recovery-mode\ndata: %s\n\n", mode)

	return tavern.StreamSSE(
		ctx,
		c.Response(),
		fanIn,
		func(s string) string { return s },
		tavern.WithStreamSnapshot(func() string { return modeFrame }),
		tavern.WithStreamHeartbeat(15*time.Second),
	)
}

// startPublisher emits replay and live events on a slow timer so the demo
// always has fresh data without overwhelming the page.
func (r *recoveryRoutes) startPublisher(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Replay event with stable ID for Last-Event-ID recovery.
			id, seq, ts := r.lab.NextReplayEvent()
			html := renderRecoveryReplayEvent(seq, id, ts.Format("15:04:05.000"), false)
			msg := tavern.NewSSEMessage("recovery-replay", html).WithID(id).String()
			r.broker.PublishWithID(topicRecoveryReplay, id, msg)

			// Live event with no replay.
			liveSeq, liveTS := r.lab.NextLiveEvent()
			liveHTML := renderRecoveryLiveEvent(liveSeq, liveTS.Format("15:04:05.000"))
			r.broker.Publish(topicRecoveryLive, tavern.NewSSEMessage("recovery-live", liveHTML).String())
		}
	}
}

func renderRecoveryReplayEvent(seq int64, id, timestamp string, replayed bool) string {
	return renderToString("render recovery replay", views.RecoveryReplayEvent(seq, id, timestamp, replayed))
}

func renderRecoverySnapshotEvent(value, timestamp string, snapshot bool) string {
	return renderToString("render recovery snapshot", views.RecoverySnapshotEvent(value, timestamp, snapshot))
}

func renderRecoveryLiveEvent(seq int64, timestamp string) string {
	return renderToString("render recovery live", views.RecoveryLiveEvent(seq, timestamp))
}
