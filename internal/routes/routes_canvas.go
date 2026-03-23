// setup:feature:demo

package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

const canvasClientCookie = "canvas_client"

type canvasRoutes struct {
	canvas *demo.PixelCanvas
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initCanvasRoutes(canvas *demo.PixelCanvas, broker *ssebroker.SSEBroker) {
	cr := &canvasRoutes{canvas: canvas, broker: broker}
	ar.e.GET("/demo/canvas", cr.handleCanvasPage)
	ar.e.GET("/demo/canvas/state", cr.handleCanvasState)
	ar.e.POST("/demo/canvas/place", cr.handlePlace)
	ar.e.POST("/demo/canvas/reset", cr.handleReset)
	ar.e.GET("/sse/canvas", cr.handleCanvasSSE)

	go cr.runTicker()
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
	x, _ := strconv.Atoi(c.FormValue("x"))
	y, _ := strconv.Atoi(c.FormValue("y"))
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
	if cr.broker.HasSubscribers(ssebroker.TopicCanvasUpdate) {
		msg := ssebroker.NewSSEMessage("canvas-reset", "").String()
		cr.broker.Publish(ssebroker.TopicCanvasUpdate, msg)
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

	ch, unsub := cr.broker.Subscribe(ssebroker.TopicCanvasUpdate)
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
			fmt.Fprint(c.Response(), msg)
			flusher.Flush()
		}
	}
}

// runTicker periodically broadcasts active client info and prunes stale clients.
func (cr *canvasRoutes) runTicker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cr.canvas.PruneStale()
		cr.broadcastClients()
	}
}

func (cr *canvasRoutes) broadcastPixel(x, y int, color string) {
	if !cr.broker.HasSubscribers(ssebroker.TopicCanvasUpdate) {
		return
	}
	data, _ := json.Marshal(map[string]any{"x": x, "y": y, "color": color})
	msg := ssebroker.NewSSEMessage("pixel-update", string(data)).String()
	cr.broker.Publish(ssebroker.TopicCanvasUpdate, msg)
}

func (cr *canvasRoutes) broadcastClients() {
	if !cr.broker.HasSubscribers(ssebroker.TopicCanvasUpdate) {
		return
	}
	clients := cr.canvas.ActiveClients()
	data, _ := json.Marshal(map[string]any{"count": len(clients), "clients": clients})
	msg := ssebroker.NewSSEMessage("clients-update", string(data)).String()
	cr.broker.Publish(ssebroker.TopicCanvasUpdate, msg)
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
		SameSite: http.SameSiteLaxMode,
	})
	return id
}
