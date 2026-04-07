// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"

	"github.com/catgoose/tavern"
	"github.com/labstack/echo/v4"
)

// Topic constants for the collaborative document demo.
const (
	topicDocContent   = "doc/content"
	topicDocStats     = "doc/stats"
	topicDocSentiment = "doc/sentiment"
	topicDocHistory   = "doc/history"
)

type docRoutes struct {
	broker       *tavern.SSEBroker
	doc          *demo.SharedDocument
	publishCount *atomic.Int64
	publishBytes *atomic.Int64
}

func (ar *appRoutes) initDocRoutes(broker *tavern.SSEBroker) {
	doc := demo.NewSharedDocument()

	var publishCount atomic.Int64
	var publishBytes atomic.Int64

	d := &docRoutes{
		broker:       broker,
		doc:          doc,
		publishCount: &publishCount,
		publishBytes: &publishBytes,
	}

	// Middleware: count all publishes on doc/* topics.
	broker.UseTopics("doc/*", func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			publishCount.Add(1)
			publishBytes.Add(int64(len(msg)))
			next(t, msg)
		}
	})

	// After hooks: content changes trigger stats + sentiment + history recalculation.
	broker.After(topicDocContent, func() {
		statsHTML := renderDocStats(doc)
		broker.Publish(topicDocStats, statsHTML)

		sentimentHTML := renderDocSentiment(doc)
		broker.Publish(topicDocSentiment, sentimentHTML)

		historyHTML := renderDocHistory(doc)
		broker.Publish(topicDocHistory, historyHTML)
	})

	// OnMutate: edit POSTs trigger content publish via mutation signal.
	broker.OnMutate("document", func(evt tavern.MutationEvent) {
		content := evt.Data.(string)
		doc.Update(content)
		html := renderDocContent(doc)
		broker.Publish(topicDocContent, html)
	})

	// Define topic group for a single SSE endpoint.
	broker.DefineGroup("doc-all", []string{topicDocContent, topicDocStats, topicDocSentiment, topicDocHistory})

	ar.e.GET("/realtime/document", d.handlePage)
	ar.e.GET("/sse/document", echo.WrapHandler(broker.GroupHandler("doc-all")))
	ar.e.POST("/realtime/document/edit", d.handleEdit)
	ar.e.POST("/realtime/document/batch", d.handleBatchEdit)
	ar.e.GET("/realtime/document/stats-badge", d.handleStatsBadge)
}

func (d *docRoutes) handlePage(c echo.Context) error {
	content := d.doc.Content()
	wc := d.doc.WordCount()
	cc := d.doc.CharCount()
	sent := d.doc.Sentiment()
	revs := d.doc.Revisions()
	return handler.RenderBaseLayout(c, views.DocumentPage(content, wc, cc, sent, revs))
}

func (d *docRoutes) handleEdit(c echo.Context) error {
	content := c.FormValue("content")
	d.broker.NotifyMutate("document", tavern.MutationEvent{
		ID:   "document",
		Data: content,
	})
	return c.NoContent(http.StatusNoContent)
}

func (d *docRoutes) handleBatchEdit(c echo.Context) error {
	action := c.FormValue("action")
	newContent := d.doc.BatchEdit(action)

	batch := d.broker.Batch()
	batch.Publish(topicDocContent, renderDocContent(d.doc))
	batch.Publish(topicDocStats, renderDocStats(d.doc))
	batch.Publish(topicDocSentiment, renderDocSentiment(d.doc))
	batch.Publish(topicDocHistory, renderDocHistory(d.doc))
	batch.Flush()

	_ = newContent
	return c.NoContent(http.StatusNoContent)
}

func (d *docRoutes) handleStatsBadge(c echo.Context) error {
	count := d.publishCount.Load()
	byteCount := d.publishBytes.Load()
	html := fmt.Sprintf(
		`<span class="badge badge-ghost badge-sm font-mono">%d publishes / %s</span>`,
		count, formatBytes(byteCount),
	)
	return c.HTML(http.StatusOK, html)
}

// --- render helpers ---

func renderDocContent(doc *demo.SharedDocument) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render doc content")
	if err := views.DocContentDisplay(doc.Content()).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderDocStats(doc *demo.SharedDocument) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render doc stats")
	if err := views.DocStatsPanel(doc.WordCount(), doc.CharCount()).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderDocSentiment(doc *demo.SharedDocument) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render doc sentiment")
	if err := views.DocSentimentBadge(doc.Sentiment()).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func renderDocHistory(doc *demo.SharedDocument) string {
	buf := &bytes.Buffer{}
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "render doc history")
	if err := views.DocHistoryList(doc.Revisions()).Render(ctx, buf); err != nil {
		return ""
	}
	return buf.String()
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
