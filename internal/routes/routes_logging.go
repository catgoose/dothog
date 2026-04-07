// setup:feature:demo

package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"

	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/shared"
	"catgoose/dothog/web/views"
	"github.com/catgoose/promolog"
	"github.com/catgoose/tavern"

	"github.com/labstack/echo/v4"
)

const loggingBase = "/platform/logging"

func (ar *appRoutes) initLoggingRoutes(broker *tavern.SSEBroker) {
	// setup:feature:sse:start
	// Wire up SSE broadcasting on error trace promotion.
	if ar.repos.ReqLogStore != nil {
		ar.repos.ReqLogStore.SetOnPromote(func(summary promolog.TraceSummary) {
			broadcastErrorTrace(broker, summary)
		})
	}

	broker.SetReplayPolicy(TopicErrorTraces, 10)
	broker.SetReplayGapPolicy(TopicErrorTraces, tavern.GapFallbackToSnapshot, nil)
	ar.e.GET("/sse/error-traces", echo.WrapHandler(broker.SSEHandler(TopicErrorTraces)))
	// setup:feature:sse:end

	ar.e.GET(loggingBase, handler.HandleComponent(views.LoggingPage()))

	// Error trigger endpoints — generate real errors with contextual slog entries.
	ar.e.GET(loggingBase+"/trigger/:code", func(c echo.Context) error {
		ctx := c.Request().Context()
		log := logger.WithContext(ctx)
		code := c.Param("code")

		switch code {
		case "404":
			log.Info("Resolving resource", "resource_type", "inventory_item", "id", "item-8832")
			log.Info("Querying database", "table", "inventory_items", "query", "SELECT * FROM inventory_items WHERE id = ?", "params", "item-8832")
			log.Info("Database query completed", "rows_returned", 0, "duration_ms", 2)
			log.Warn("Resource lookup failed", "resource_type", "inventory_item", "id", "item-8832")
			return handler.HandleHypermediaError(c, http.StatusNotFound, "Resource not found",
				fmt.Errorf("get item item-8832: %w", errors.New("sql: no rows in result set")))

		case "400":
			log.Info("Parsing request body", "content_type", "application/x-www-form-urlencoded")
			log.Info("Validating request parameters", "required", []string{"name", "email", "quantity"})
			log.Warn("Validation failed", "missing_fields", []string{"email", "quantity"})
			return handler.HandleHypermediaError(c, http.StatusBadRequest, "Bad request",
				fmt.Errorf("validate order: %w", errors.New("missing required fields: email, quantity")))

		case "500":
			log.Info("Processing order", "order_id", "ORD-20260311-4471", "customer_id", "cust-229")
			log.Info("Verifying inventory", "sku", "WIDGET-PRO-X", "requested_qty", 5)
			log.Info("Inventory check passed", "sku", "WIDGET-PRO-X", "available_qty", 12)
			log.Info("Initiating payment", "provider", "stripe", "amount_cents", 14995)
			log.Info("Payment authorized", "charge_id", "ch_3Q7xK2LkdIwHu7ix", "duration_ms", 843)
			log.Error("Database write failed", "error", "pq: deadlock detected", "table", "orders", "retry_attempt", 1)
			log.Error("Database write failed after retry", "error", "pq: deadlock detected", "retry_attempt", 2)
			log.Info("Payment rollback completed", "refund_id", "re_3Q7xK9LkdIwHu7ix")
			return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Internal server error",
				fmt.Errorf("process order ORD-20260311-4471: commit: %w", errors.New("pq: deadlock detected")))

		case "403":
			log.Info("Authenticating user", "session_id", "sess-a8c3e9f1")
			log.Info("User authenticated", "user_id", "usr-1042", "roles", []string{"viewer"})
			log.Warn("Authorization denied", "resource", "/admin/settings", "required", "admin", "actual", "viewer")
			return handler.HandleHypermediaError(c, http.StatusForbidden, "Forbidden",
				fmt.Errorf("authorize /admin/settings: %w", errors.New("role viewer cannot access admin resource")))

		default:
			return handler.HandleHypermediaError(c, http.StatusBadRequest, "Unknown error code",
				fmt.Errorf("unsupported code: %s", code))
		}
	})

	// List recent traces
	ar.e.GET(loggingBase+"/traces", func(c echo.Context) error {
		if ar.repos.ReqLogStore == nil {
			return handler.RenderComponent(c, views.LoggingTracesList(nil))
		}
		traces, _, err := ar.repos.ReqLogStore.ListTraces(c.Request().Context(), promolog.TraceFilter{
			Sort: "CreatedAt", Dir: "desc", Page: 1, PerPage: 20,
		})
		if err != nil {
			return handler.HandleHypermediaError(c, 500, "Failed to load traces", err)
		}
		return handler.RenderComponent(c, views.LoggingTracesList(traces))
	})

	// Simulate support report — returns formatted JSON of what IssueReporter would receive.
	ar.e.GET(loggingBase+"/report/:requestID", func(c echo.Context) error {
		requestID := c.Param("requestID")
		if ar.repos.ReqLogStore == nil {
			return handler.HandleHypermediaError(c, 404, "Store not configured", nil)
		}
		trace, err := ar.repos.ReqLogStore.Get(c.Request().Context(), requestID)
		if err != nil {
			logger.WithContext(c.Request().Context()).Error("Failed to retrieve error trace",
				"request_id", requestID, "error", err)
		}
		if trace == nil {
			return handler.HandleHypermediaError(c, 404, "Trace not found or expired", nil)
		}

		payload := map[string]any{
			"request_id":  trace.RequestID,
			"error_chain": trace.ErrorChain,
			"status_code": trace.StatusCode,
			"route":       trace.Route,
			"method":      trace.Method,
			"user_agent":  trace.UserAgent,
			"remote_ip":   trace.RemoteIP,
			"user_id":     trace.UserID,
			"created_at":  trace.CreatedAt,
			"log_entries": trace.Entries,
		}
		jsonBytes, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return handler.HandleHypermediaError(c, 500, "Failed to marshal report", err)
		}
		return handler.RenderComponent(c, views.LoggingReportOutput(trace, string(jsonBytes)))
	})
}

// setup:feature:sse:start

var traceCounter atomic.Int64

func broadcastErrorTrace(broker *tavern.SSEBroker, summary promolog.TraceSummary) {
	if !broker.HasSubscribers(TopicErrorTraces) {
		return
	}
	buf := new(bytes.Buffer)
	ctx := shared.WithContextIDAndDescription(context.Background(), shared.GenerateContextID(), "broadcast error trace")
	if err := views.LoggingTraceRowOOB(summary).Render(ctx, buf); err != nil {
		return
	}
	eventID := fmt.Sprintf("et%d", traceCounter.Add(1))
	msg := tavern.NewSSEMessage("error-trace", buf.String()).
		WithID(eventID).
		String()
	broker.PublishWithID(TopicErrorTraces, eventID, msg)
}

// setup:feature:sse:end
