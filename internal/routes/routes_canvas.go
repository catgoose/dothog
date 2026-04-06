// setup:feature:demo

package routes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"time"

	appenv "catgoose/dothog/internal/env"
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"github.com/catgoose/tavern"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

const canvasClientCookie = "canvas_client"

type canvasRoutes struct {
	canvas *demo.PixelCanvas
	broker *tavern.SSEBroker
	ctx    context.Context
}

func (ar *appRoutes) initCanvasRoutes(canvas *demo.PixelCanvas, broker *tavern.SSEBroker) {
	cr := &canvasRoutes{canvas: canvas, broker: broker, ctx: ar.ctx}
	ar.e.GET("/realtime/canvas", cr.handleCanvasPage)
	ar.e.GET("/realtime/canvas/state", cr.handleCanvasState)
	ar.e.POST("/realtime/canvas/place", cr.handlePlace)
	ar.e.POST("/realtime/canvas/reset", cr.handleReset)
	ar.e.GET("/sse/canvas", cr.handleCanvasSSE)

	broker.SetOrdered(TopicCanvasUpdate, true)

	broker.RunPublisher(ar.ctx, func(ctx context.Context) {
		cr.runTicker()
	})
}

func (cr *canvasRoutes) handleCanvasPage(c echo.Context) error {
	getOrCreateClientID(c)
	// Exclude white (#ffffff) from random start color since it's invisible on the
	// white canvas background. White stays in the palette as an eraser.
	nonWhite := make([]string, 0, len(demo.CanvasPalette))
	for _, c := range demo.CanvasPalette {
		if c != "#ffffff" {
			nonWhite = append(nonWhite, c)
		}
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(nonWhite))))
	startColor := nonWhite[n.Int64()]
	return handler.RenderBaseLayout(c, views.CanvasPage(startColor))
}

func (cr *canvasRoutes) handleCanvasState(c echo.Context) error {
	cells := cr.canvas.Snapshot()
	return c.JSON(http.StatusOK, cells[:])
}

func (cr *canvasRoutes) handlePlace(c echo.Context) error {
	clientID := getOrCreateClientID(c)
	x, err := strconv.Atoi(c.FormValue("x"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid x coordinate")
	}
	y, err := strconv.Atoi(c.FormValue("y"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid y coordinate")
	}
	color := c.FormValue("color")
	if err := cr.canvas.PlaceColor(x, y, color); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	cr.canvas.TouchClient(clientID, color)
	cr.broadcastPixel(x, y, color)
	return c.NoContent(http.StatusOK)
}

func (cr *canvasRoutes) handleReset(c echo.Context) error {
	cr.canvas.Reset()
	if cr.broker.HasSubscribers(TopicCanvasUpdate) {
		msg := tavern.NewSSEMessage("canvas-reset", "").String()
		cr.broker.Publish(TopicCanvasUpdate, msg)
	}
	return c.NoContent(http.StatusOK)
}

func (cr *canvasRoutes) handleCanvasSSE(c echo.Context) error {
	clientID := getOrCreateClientID(c)
	color := c.QueryParam("color")
	if color == "" {
		color = demo.CanvasPalette[0]
	}
	cr.canvas.TouchClient(clientID, color)

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ch, unsub := cr.broker.Subscribe(TopicCanvasUpdate)
	defer unsub()

	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			cr.canvas.TouchClient(clientID, color)
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			_, _ = fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

// runTicker periodically broadcasts active client info and prunes stale clients.
func (cr *canvasRoutes) runTicker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-cr.ctx.Done():
			return
		case <-ticker.C:
			cr.canvas.PruneStale()
			cr.broadcastClients()
		}
	}
}

func (cr *canvasRoutes) broadcastPixel(x, y int, color string) {
	if !cr.broker.HasSubscribers(TopicCanvasUpdate) {
		return
	}
	data, err := json.Marshal(map[string]any{"x": x, "y": y, "color": color})
	if err != nil {
		slog.Error("marshal pixel update", "error", err)
		return
	}
	msg := tavern.NewSSEMessage("pixel-update", string(data)).String()
	cr.broker.Publish(TopicCanvasUpdate, msg)
}

func (cr *canvasRoutes) broadcastClients() {
	if !cr.broker.HasSubscribers(TopicCanvasUpdate) {
		return
	}
	clients := cr.canvas.ActiveClients()
	data, err := json.Marshal(map[string]any{"count": len(clients), "clients": clients})
	if err != nil {
		slog.Error("marshal clients update", "error", err)
		return
	}
	msg := tavern.NewSSEMessage("clients-update", string(data)).String()
	cr.broker.Publish(TopicCanvasUpdate, msg)
}

func getOrCreateClientID(c echo.Context) string {
	if cookie, err := c.Cookie(canvasClientCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	c.SetCookie(&http.Cookie{
		Name:     canvasClientCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   !appenv.Dev(),
		SameSite: http.SameSiteLaxMode,
	})
	return id
}
