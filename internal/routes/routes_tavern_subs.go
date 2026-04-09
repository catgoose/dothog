// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	appenv "catgoose/dothog/internal/env"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

const subsGroupCookie = "tavern_subs_group"

// subsTopics contains all the topics used by the subscription lab.
var subsTopics = struct {
	scoped    string
	data      []string
	multi     []string
	vip       []string
	standard  []string
}{
	scoped: "tavern/subs/scoped",
	data: []string{
		"tavern/subs/data/temp",
		"tavern/subs/data/humidity",
		"tavern/subs/data/pressure",
	},
	multi: []string{
		"tavern/subs/multi/alpha",
		"tavern/subs/multi/beta",
		"tavern/subs/multi/gamma",
	},
	vip:      []string{"tavern/subs/group/vip-alerts", "tavern/subs/group/vip-deals"},
	standard: []string{"tavern/subs/group/general"},
}

type tavernSubsRoutes struct {
	broker *tavern.SSEBroker
}

func (ar *appRoutes) initTavernSubsRoutes(broker *tavern.SSEBroker) {
	s := &tavernSubsRoutes{broker: broker}

	broker.DynamicGroup("tavern-subs-dynamic", dynamicGroupFromCookie(
		subsGroupCookie,
		subsTopics.standard,
		func(value string) []string {
			if value == "vip" {
				return subsTopics.vip
			}
			return subsTopics.standard
		},
	))

	ar.e.GET("/realtime/tavern/subscriptions", s.handlePage)
	ar.e.GET("/realtime/tavern/subscriptions/scoped-panel", s.handleScopedPanel)
	ar.e.GET("/realtime/tavern/subscriptions/glob-panel", s.handleGlobPanel)
	ar.e.GET("/sse/tavern/subs/scoped", s.handleScopedSSE)
	ar.e.GET("/sse/tavern/subs/glob", s.handleGlobSSE)
	ar.e.GET("/sse/tavern/subs/multi", s.handleMultiSSE)
	ar.e.GET("/sse/tavern/subs/dynamic", echo.WrapHandler(broker.DynamicGroupHandler("tavern-subs-dynamic")))
	ar.e.POST("/realtime/tavern/subscriptions/group", s.handleGroupSwitch)

	broker.RunPublisher(ar.ctx, s.startPublisher)
}

func (s *tavernSubsRoutes) handlePage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.TavernSubsPage())
}

func (s *tavernSubsRoutes) handleScopedPanel(c echo.Context) error {
	scope := c.QueryParam("scope")
	if scope == "" {
		scope = "scope-a"
	}
	return handler.RenderComponent(c, views.SubsScopedPanel(scope))
}

func (s *tavernSubsRoutes) handleGlobPanel(c echo.Context) error {
	pattern := c.QueryParam("pattern")
	if pattern == "" {
		pattern = "tavern/subs/data/**"
	}
	return handler.RenderComponent(c, views.SubsGlobPanel(pattern))
}

func (s *tavernSubsRoutes) handleGroupSwitch(c echo.Context) error {
	group := c.FormValue("group")
	c.SetCookie(&http.Cookie{
		Name:     subsGroupCookie,
		Value:    group,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: false,
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
	return handler.RenderComponent(c, views.SubsDynamicPanel())
}

func (s *tavernSubsRoutes) handleScopedSSE(c echo.Context) error {
	scope := c.QueryParam("scope")
	if scope == "" {
		scope = "scope-a"
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := s.broker.SubscribeScoped(subsTopics.scoped, scope)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

func (s *tavernSubsRoutes) handleGlobSSE(c echo.Context) error {
	pattern := c.QueryParam("pattern")
	if pattern == "" {
		pattern = "tavern/subs/data/**"
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := s.broker.SubscribeGlob(pattern)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case tm, ok := <-ch:
			if !ok {
				return nil
			}
			msg := tavern.NewSSEMessage("glob-events", tm.Data).String()
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

func (s *tavernSubsRoutes) handleMultiSSE(c echo.Context) error {
	topicParam := c.QueryParam("topics")
	topics := strings.Split(topicParam, ",")
	if len(topics) == 0 || (len(topics) == 1 && topics[0] == "") {
		topics = subsTopics.multi
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := s.broker.SubscribeMulti(topics...)
	defer unsub()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case tm, ok := <-ch:
			if !ok {
				return nil
			}
			msg := tavern.NewSSEMessage("multi-events", tm.Data).String()
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

func (s *tavernSubsRoutes) startPublisher(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	seq := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			seq++
			ts := time.Now().Format("15:04:05")

			// Scoped publishes: different message per scope.
			for _, scope := range []string{"scope-a", "scope-b"} {
				html := renderSubsEvent(subsTopics.scoped, fmt.Sprintf("[%s] event #%d", scope, seq), ts)
				msg := tavern.NewSSEMessage("scoped-events", html).String()
				s.broker.PublishTo(subsTopics.scoped, scope, msg)
			}

			// Glob-matchable data topics.
			for _, topic := range subsTopics.data {
				short := topic[strings.LastIndex(topic, "/")+1:]
				val := fmt.Sprintf("%.1f", 20.0+rand.Float64()*10)
				html := renderSubsEvent(topic, fmt.Sprintf("%s=%s", short, val), ts)
				s.broker.Publish(topic, html)
			}

			// Multi topics.
			for _, topic := range subsTopics.multi {
				short := topic[strings.LastIndex(topic, "/")+1:]
				html := renderSubsEvent(topic, fmt.Sprintf("%s tick #%d", short, seq), ts)
				msg := tavern.NewSSEMessage("multi-events", html).String()
				s.broker.Publish(topic, msg)
			}

			// Dynamic group topics.
			for _, topic := range append(subsTopics.vip, subsTopics.standard...) {
				short := topic[strings.LastIndex(topic, "/")+1:]
				html := renderSubsEvent(topic, fmt.Sprintf("%s #%d", short, seq), ts)
				s.broker.Publish(topic, html)
			}
		}
	}
}

func renderSubsEvent(topic, message, timestamp string) string {
	return renderToString("render subs event", views.SubsEventEntry(topic, message, timestamp))
}
