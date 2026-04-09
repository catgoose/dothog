// setup:feature:demo

package routes

import (
	"fmt"
	"net/http"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
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
	broker   *tavern.SSEBroker
	doc      *demo.SharedDocument
	pubStats *demo.PublishStats
}

func (ar *appRoutes) initDocRoutes(broker *tavern.SSEBroker) {
	doc := demo.NewSharedDocument()

	d := &docRoutes{
		broker:   broker,
		doc:      doc,
		pubStats: &demo.PublishStats{},
	}

	// Middleware: count all publishes on doc/* topics.
	broker.UseTopics("doc/*", func(next tavern.PublishFunc) tavern.PublishFunc {
		return func(t, msg string) {
			d.pubStats.Add(len(msg))
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
	count, byteCount := d.pubStats.Snapshot()
	html := fmt.Sprintf(
		`<span class="badge badge-ghost badge-sm font-mono">%d publishes / %s</span>`,
		count, formatBytes(byteCount),
	)
	return c.HTML(http.StatusOK, html)
}

// --- render helpers ---

func renderDocContent(doc *demo.SharedDocument) string {
	return renderToString("render doc content", views.DocContentDisplay(doc.Content()))
}

func renderDocStats(doc *demo.SharedDocument) string {
	return renderToString("render doc stats", views.DocStatsPanel(doc.WordCount(), doc.CharCount()))
}

func renderDocSentiment(doc *demo.SharedDocument) string {
	return renderToString("render doc sentiment", views.DocSentimentBadge(doc.Sentiment()))
}

func renderDocHistory(doc *demo.SharedDocument) string {
	return renderToString("render doc history", views.DocHistoryList(doc.Revisions()))
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
