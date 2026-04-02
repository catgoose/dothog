// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

const labBase = hypermediaBase + "/lab"

const (
	mbWidth   = 120
	mbHeight  = 60
	mbMaxIter = 256
)

var mbPalette = [16][3]uint8{
	{66, 30, 15}, {25, 7, 26}, {9, 1, 47}, {4, 4, 73},
	{0, 7, 100}, {12, 44, 138}, {24, 82, 177}, {57, 125, 209},
	{134, 181, 229}, {211, 236, 248}, {241, 233, 191}, {248, 212, 120},
	{232, 167, 53}, {200, 117, 17}, {159, 74, 4}, {106, 27, 4},
}

var mandelbrotState struct {
	cancel context.CancelFunc
	mu     sync.Mutex
}

func (ar *appRoutes) initLabRoutes(broker *tavern.SSEBroker) {
	ar.e.GET(labBase, handleLabPage)
	ar.e.POST(labBase+"/mandelbrot/start", handleMandelbrotStart(ar.ctx, broker))
	ar.e.POST(labBase+"/mandelbrot/reset", handleMandelbrotReset())
	ar.e.GET("/sse/lab", handleSSELab(broker))
}

func handleLabPage(c echo.Context) error {
	return handler.RenderBaseLayout(c, views.LabPage())
}

func handleSSELab(broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return fmt.Errorf("streaming not supported")
		}
		flusher.Flush()

		ch, unsub := broker.Subscribe(TopicLabMandelbrot)
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
				fmt.Fprint(c.Response(), msg)
				flusher.Flush()
			}
		}
	}
}

func handleMandelbrotStart(appCtx context.Context, broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		mandelbrotState.mu.Lock()
		if mandelbrotState.cancel != nil {
			mandelbrotState.cancel()
		}
		ctx, cancel := context.WithCancel(appCtx)
		mandelbrotState.cancel = cancel
		mandelbrotState.mu.Unlock()

		go publishMandelbrot(ctx, broker)

		return handler.RenderComponent(c, views.MandelbrotRunning())
	}
}

func handleMandelbrotReset() echo.HandlerFunc {
	return func(c echo.Context) error {
		mandelbrotState.mu.Lock()
		if mandelbrotState.cancel != nil {
			mandelbrotState.cancel()
			mandelbrotState.cancel = nil
		}
		mandelbrotState.mu.Unlock()

		return handler.RenderComponent(c, views.MandelbrotReady())
	}
}

func mbIter(cr, ci float64) (int, float64, float64) {
	zr, zi := 0.0, 0.0
	for i := 0; i < mbMaxIter; i++ {
		zr2, zi2 := zr*zr, zi*zi
		if zr2+zi2 > 4.0 {
			return i, zr, zi
		}
		zi = 2*zr*zi + ci
		zr = zr2 - zi2 + cr
	}
	return mbMaxIter, zr, zi
}

func mbColor(iter int, zr, zi float64) string {
	if iter == mbMaxIter {
		return "#000"
	}
	mu := float64(iter) + 1.0 - math.Log2(math.Log(zr*zr+zi*zi)/2.0)
	idx := int(math.Abs(mu*0.7)) % len(mbPalette)
	c := mbPalette[idx]
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
}

func publishMandelbrot(ctx context.Context, broker *tavern.SSEBroker) {
	realMin, realMax := -2.5, 1.0
	imagMin, imagMax := -1.1, 1.1

	totalPixels := 0

	for row := 0; row < mbHeight; row++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for subscribers
		for !broker.HasSubscribers(TopicLabMandelbrot) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}

		ci := imagMin + (float64(row)+0.5)/float64(mbHeight)*(imagMax-imagMin)

		var rowBuf strings.Builder
		rowBuf.WriteString(`<div class="leading-none whitespace-nowrap">`)

		maxIterInRow := 0
		for col := 0; col < mbWidth; col++ {
			cr := realMin + (float64(col)+0.5)/float64(mbWidth)*(realMax-realMin)
			iter, zr, zi := mbIter(cr, ci)
			color := mbColor(iter, zr, zi)
			if iter > maxIterInRow {
				maxIterInRow = iter
			}
			fmt.Fprintf(&rowBuf, `<span style="color:%s">█</span>`, color)
		}
		rowBuf.WriteString(`</div>`)

		totalPixels += mbWidth

		var oob strings.Builder
		// Canvas row append
		fmt.Fprintf(&oob, `<div hx-swap-oob="beforeend:#mandelbrot-canvas">%s</div>`, rowBuf.String())
		// Status update
		fmt.Fprintf(&oob, `<div id="mb-status" hx-swap-oob="innerHTML">Row %d/%d · %s pixels</div>`, row+1, mbHeight, formatCommas(totalPixels))

		msg := tavern.NewSSEMessage("lab-mandelbrot", oob.String()).String()
		broker.Publish(TopicLabMandelbrot, msg)

		// Variable delay: 50ms base + maxIterInRow/4 ms, capped at 150ms
		delay := 50 + maxIterInRow/4
		if delay > 150 {
			delay = 150
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(delay) * time.Millisecond):
		}
	}

	// Completion message
	totalPixels = mbWidth * mbHeight
	var done strings.Builder
	fmt.Fprintf(&done, `<div id="mb-status" hx-swap-oob="innerHTML">✓ Complete · %s pixels rendered</div>`, formatCommas(totalPixels))
	done.WriteString(`<div id="mb-controls" hx-swap-oob="innerHTML"><button class="btn btn-sm btn-outline btn-error" hx-post="/hypermedia/lab/mandelbrot/reset" hx-target="#mb-controls" hx-swap="outerHTML">Reset</button></div>`)

	msg := tavern.NewSSEMessage("lab-mandelbrot", done.String()).String()
	broker.Publish(TopicLabMandelbrot, msg)
}

func formatCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
