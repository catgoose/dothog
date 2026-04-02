// setup:feature:demo

package routes

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
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
	mbWidth        = 120
	mbHeight       = 60
	mbMaxIter      = 256
	mbDefaultDelay = 2 * time.Second
	mbDefaultDepth = 30
)

var mbPalette = [16][3]uint8{
	{66, 30, 15}, {25, 7, 26}, {9, 1, 47}, {4, 4, 73},
	{0, 7, 100}, {12, 44, 138}, {24, 82, 177}, {57, 125, 209},
	{134, 181, 229}, {211, 236, 248}, {241, 233, 191}, {248, 212, 120},
	{232, 167, 53}, {200, 117, 17}, {159, 74, 4}, {106, 27, 4},
}

type mbViewport struct {
	realMin, realMax, imagMin, imagMax float64
}

var mbDefaultVP = mbViewport{
	realMin: -2.5, realMax: 1.0,
	imagMin: -1.1, imagMax: 1.1,
}

var mandelbrotState struct {
	cancel   context.CancelFunc
	delay    time.Duration
	maxDepth int
	mu       sync.Mutex
}

func (ar *appRoutes) initLabRoutes(broker *tavern.SSEBroker) {
	mandelbrotState.delay = mbDefaultDelay
	mandelbrotState.maxDepth = mbDefaultDepth
	ar.e.GET(labBase, handleLabPage(ar.ctx, broker))
	ar.e.POST(labBase+"/mandelbrot/settings", handleMandelbrotSettings(ar.ctx, broker))
	ar.e.POST(labBase+"/mandelbrot/reset", handleMandelbrotReset(ar.ctx, broker))
	ar.e.GET("/sse/lab", handleSSELab(broker))
}

// handleLabPage pre-renders the default view and starts the auto-zoom publisher.
func handleLabPage(appCtx context.Context, broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		grid, _ := renderMandelbrotGrid(mbDefaultVP, mbMaxIter)

		mandelbrotState.mu.Lock()
		if mandelbrotState.cancel != nil {
			mandelbrotState.cancel()
		}
		ctx, cancel := context.WithCancel(appCtx)
		mandelbrotState.cancel = cancel
		mandelbrotState.mu.Unlock()

		go publishAutoZoom(ctx, broker)

		return handler.RenderBaseLayout(c, views.LabPage(grid))
	}
}

// handleMandelbrotSettings applies new delay/depth and restarts the zoom.
func handleMandelbrotSettings(appCtx context.Context, broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		delay, _ := strconv.ParseFloat(c.FormValue("delay"), 64)
		if delay < 0.5 {
			delay = 0.5
		} else if delay > 5.0 {
			delay = 5.0
		}
		maxDepth, _ := strconv.Atoi(c.FormValue("max_depth"))
		if maxDepth < 5 {
			maxDepth = 5
		} else if maxDepth > 50 {
			maxDepth = 50
		}

		mandelbrotState.mu.Lock()
		mandelbrotState.delay = time.Duration(delay * float64(time.Second))
		mandelbrotState.maxDepth = maxDepth
		if mandelbrotState.cancel != nil {
			mandelbrotState.cancel()
		}
		ctx, cancel := context.WithCancel(appCtx)
		mandelbrotState.cancel = cancel
		mandelbrotState.mu.Unlock()

		grid, _ := renderMandelbrotGrid(mbDefaultVP, mbMaxIter)
		go publishAutoZoom(ctx, broker)

		return handler.RenderComponent(c, views.MandelbrotSettingsApplied(grid, maxDepth))
	}
}

// handleMandelbrotReset restores defaults and restarts from the beginning.
func handleMandelbrotReset(appCtx context.Context, broker *tavern.SSEBroker) echo.HandlerFunc {
	return func(c echo.Context) error {
		mandelbrotState.mu.Lock()
		mandelbrotState.delay = mbDefaultDelay
		mandelbrotState.maxDepth = mbDefaultDepth
		if mandelbrotState.cancel != nil {
			mandelbrotState.cancel()
		}
		ctx, cancel := context.WithCancel(appCtx)
		mandelbrotState.cancel = cancel
		mandelbrotState.mu.Unlock()

		grid, _ := renderMandelbrotGrid(mbDefaultVP, mbMaxIter)
		go publishAutoZoom(ctx, broker)

		return handler.RenderComponent(c, views.MandelbrotResetResponse(grid))
	}
}

// handleSSELab streams SSE messages for the lab page.
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

// ── Auto-zoom publisher ─────────────────────────────────────────────────────

func publishAutoZoom(ctx context.Context, broker *tavern.SSEBroker) {
	// Wait for SSE subscriber
	for !broker.HasSubscribers(TopicLabMandelbrot) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}

	mandelbrotState.mu.Lock()
	maxDepth := mandelbrotState.maxDepth
	mandelbrotState.mu.Unlock()

	vp := mbDefaultVP
	maxIter := mbMaxIter

	// Compute iteration matrix for default view (already rendered on page, don't publish)
	_, iters := renderMandelbrotGrid(vp, maxIter)

	for depth := 1; depth <= maxDepth; depth++ {
		// Find interesting point and zoom
		col, row := findInterestingPoint(iters, maxIter)
		cr := vp.realMin + (float64(col)+0.5)/float64(mbWidth)*(vp.realMax-vp.realMin)
		ci := vp.imagMin + (float64(row)+0.5)/float64(mbHeight)*(vp.imagMax-vp.imagMin)

		// Zoom 2x centered on interesting point
		rHalf := (vp.realMax - vp.realMin) / 4
		iHalf := (vp.imagMax - vp.imagMin) / 4
		vp = mbViewport{
			realMin: cr - rHalf,
			realMax: cr + rHalf,
			imagMin: ci - iHalf,
			imagMax: ci + iHalf,
		}
		maxIter = mbMaxIter + depth*64

		// Read current delay from state (allows live adjustment)
		mandelbrotState.mu.Lock()
		delay := mandelbrotState.delay
		mandelbrotState.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Compute zoomed view
		grid, newIters := renderMandelbrotGrid(vp, maxIter)
		iters = newIters

		// Publish frame via SSE (canvas + status only, controls stay as-is)
		var buf strings.Builder
		buf.WriteString(`<div id="mandelbrot-canvas" hx-swap-oob="innerHTML">`)
		buf.WriteString(grid)
		buf.WriteString(`</div>`)
		zoomMult := 1 << uint(depth)
		fmt.Fprintf(&buf, `<div id="mb-status" hx-swap-oob="innerHTML"><span class="font-mono">×%d</span> · depth %d/%d</div>`, zoomMult, depth, maxDepth)

		msg := tavern.NewSSEMessage("lab-mandelbrot", buf.String()).String()
		broker.Publish(TopicLabMandelbrot, msg)
	}

	// Reached max depth
	var done strings.Builder
	fmt.Fprintf(&done, `<div id="mb-status" hx-swap-oob="innerHTML"><span class="font-mono">×%d</span> · maximum depth reached</div>`, 1<<uint(maxDepth))
	msg := tavern.NewSSEMessage("lab-mandelbrot", done.String()).String()
	broker.Publish(TopicLabMandelbrot, msg)
}

// ── Mandelbrot computation ──────────────────────────────────────────────────

func mbIterN(cr, ci float64, maxIter int) (int, float64, float64) {
	zr, zi := 0.0, 0.0
	for i := 0; i < maxIter; i++ {
		zr2, zi2 := zr*zr, zi*zi
		if zr2+zi2 > 4.0 {
			return i, zr, zi
		}
		zi = 2*zr*zi + ci
		zr = zr2 - zi2 + cr
	}
	return maxIter, zr, zi
}

func mbColorN(iter, maxIter int, zr, zi float64) string {
	if iter == maxIter {
		return "#000"
	}
	mu := float64(iter) + 1.0 - math.Log2(math.Log(zr*zr+zi*zi)/2.0)
	idx := int(math.Abs(mu*0.7)) % len(mbPalette)
	c := mbPalette[idx]
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
}

// renderMandelbrotGrid computes the full grid HTML and the iteration matrix.
func renderMandelbrotGrid(vp mbViewport, maxIter int) (string, [][]int) {
	iters := make([][]int, mbHeight)
	var buf strings.Builder
	buf.Grow(mbWidth * mbHeight * 60)
	fmt.Fprintf(&buf, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" style="display:block;width:100%%">`, mbWidth, mbHeight)

	for row := 0; row < mbHeight; row++ {
		ci := vp.imagMin + (float64(row)+0.5)/float64(mbHeight)*(vp.imagMax-vp.imagMin)
		iters[row] = make([]int, mbWidth)
		for col := 0; col < mbWidth; col++ {
			cr := vp.realMin + (float64(col)+0.5)/float64(mbWidth)*(vp.realMax-vp.realMin)
			iter, zr, zi := mbIterN(cr, ci, maxIter)
			iters[row][col] = iter
			color := mbColorN(iter, maxIter, zr, zi)
			fmt.Fprintf(&buf, `<rect x="%d" y="%d" width="1" height="1" fill="%s"/>`, col, row, color)
		}
	}

	buf.WriteString("</svg>")
	return buf.String(), iters
}

// findInterestingPoint finds the region with the highest boundary density.
// It divides the grid into blocks, scores each by the variance of iteration
// counts (high variance = boundary region), and picks randomly from the top 3.
func findInterestingPoint(iters [][]int, maxIter int) (int, int) {
	type candidate struct {
		col, row int
		score    float64
	}

	blockW, blockH := 20, 10
	var candidates []candidate

	for by := 0; by <= mbHeight-blockH; by += blockH / 2 {
		for bx := 0; bx <= mbWidth-blockW; bx += blockW / 2 {
			sum, count := 0.0, 0
			for y := by; y < by+blockH && y < mbHeight; y++ {
				for x := bx; x < bx+blockW && x < mbWidth; x++ {
					iter := iters[y][x]
					if iter > 0 && iter < maxIter {
						sum += float64(iter)
						count++
					}
				}
			}
			// Require at least 25% non-trivial pixels for a meaningful score
			total := blockW * blockH
			if count < total/4 {
				continue
			}
			mean := sum / float64(count)

			// Compute variance (high variance = boundary = interesting)
			variance := 0.0
			for y := by; y < by+blockH && y < mbHeight; y++ {
				for x := bx; x < bx+blockW && x < mbWidth; x++ {
					iter := iters[y][x]
					if iter > 0 && iter < maxIter {
						d := float64(iter) - mean
						variance += d * d
					}
				}
			}
			variance /= float64(count)

			candidates = append(candidates, candidate{
				col:   bx + blockW/2,
				row:   by + blockH/2,
				score: variance,
			})
		}
	}

	if len(candidates) == 0 {
		return mbWidth / 2, mbHeight / 2
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Pick randomly from top 3 for variety
	top := 3
	if top > len(candidates) {
		top = len(candidates)
	}
	pick := candidates[rand.IntN(top)]
	return pick.col, pick.row
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
