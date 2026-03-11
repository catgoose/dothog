// setup:feature:demo

package routes

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"catgoose/harmony/internals/logger"
	"catgoose/harmony/internals/routes/handler"
	"catgoose/harmony/internals/routes/hypermedia"
	"catgoose/harmony/internals/routes/middleware"
	"catgoose/harmony/internals/routes/response"
	"catgoose/harmony/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initErrorsRoutes() {
	base := hypermediaBase + "/errors"
	var flakyCount int64

	ar.e.GET(base, handler.HandleComponent(views.ErrorsPage()))

	// Banner error triggers — return error via HandleHypermediaError so the
	// middleware renders into #error-status via OOB swap.
	ar.e.GET(base+"/trigger/:code", func(c echo.Context) error {
		ctx := c.Request().Context()
		log := logger.WithContext(ctx)
		code := c.Param("code")

		switch code {
		case "404":
			log.Info("Resolving resource", "resource_type", "inventory_item", "id", "item-8832")
			log.Info("Querying database", "table", "inventory_items", "query", "SELECT * FROM inventory_items WHERE id = ?", "params", "item-8832")
			log.Info("Database query completed", "table", "inventory_items", "rows_returned", 0, "duration_ms", 2)
			log.Warn("Resource lookup failed", "resource_type", "inventory_item", "id", "item-8832")
			return handler.HandleHypermediaError(c, http.StatusNotFound, "Resource not found", errors.New("the requested item does not exist"))

		case "400":
			log.Info("Parsing request body", "content_type", c.Request().Header.Get("Content-Type"))
			log.Info("Validating request parameters", "required", []string{"name", "email", "quantity"})
			log.Warn("Validation failed", "missing_fields", []string{"email", "quantity"}, "provided_fields", []string{"name"})
			return handler.HandleHypermediaError(c, http.StatusBadRequest, "Bad request", errors.New("missing required parameter"))

		case "500":
			log.Info("Processing order", "order_id", "ORD-20260311-4471", "customer_id", "cust-229")
			log.Info("Verifying inventory", "sku", "WIDGET-PRO-X", "requested_qty", 5)
			log.Info("Inventory check passed", "sku", "WIDGET-PRO-X", "available_qty", 12)
			log.Info("Initiating payment", "provider", "stripe", "amount_cents", 14995, "currency", "USD")
			log.Info("Payment authorized", "provider", "stripe", "charge_id", "ch_3Q7xK2LkdIwHu7ix", "duration_ms", 843)
			log.Info("Committing order to database", "order_id", "ORD-20260311-4471")
			log.Error("Database write failed", "error", "pq: deadlock detected", "table", "orders", "operation", "INSERT", "retry_attempt", 1)
			log.Error("Database write failed after retry", "error", "pq: deadlock detected", "table", "orders", "operation", "INSERT", "retry_attempt", 2)
			log.Error("Initiating payment rollback", "provider", "stripe", "charge_id", "ch_3Q7xK2LkdIwHu7ix")
			log.Info("Payment rollback completed", "provider", "stripe", "charge_id", "ch_3Q7xK2LkdIwHu7ix", "refund_id", "re_3Q7xK9LkdIwHu7ix")
			return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Internal server error", errors.New("unexpected failure in processing"))

		case "403":
			log.Info("Authenticating user", "session_id", "sess-a8c3e9f1", "method", "bearer_token")
			log.Info("User authenticated", "user_id", "usr-1042", "email", "jdoe@example.com", "roles", []string{"viewer"})
			log.Info("Checking authorization", "resource", "/admin/settings", "required_role", "admin", "user_roles", []string{"viewer"})
			log.Warn("Authorization denied", "user_id", "usr-1042", "resource", "/admin/settings", "reason", "insufficient_role", "required", "admin", "actual", "viewer")
			return handler.HandleHypermediaError(c, http.StatusForbidden, "Forbidden", errors.New("you do not have permission to access this resource"))

		default:
			log.Warn("Unknown error code requested", "code", code)
			return handler.HandleHypermediaError(c, http.StatusBadRequest, "Unknown error code", fmt.Errorf("unsupported code: %s", code))
		}
	})

	// Inline form errors — validates name and email, returns 422 with inline errors or success.
	ar.e.POST(base+"/form", func(c echo.Context) error {
		name := strings.TrimSpace(c.FormValue("name"))
		email := strings.TrimSpace(c.FormValue("email"))

		var errs []string
		if name == "" {
			errs = append(errs, "Name is required")
		}
		if !strings.Contains(email, "@") {
			errs = append(errs, "Email must contain @")
		}

		if len(errs) > 0 {
			c.Response().Status = http.StatusUnprocessableEntity
			return handler.RenderComponent(c, views.ErrorsFormError(errs))
		}
		return handler.RenderComponent(c, views.ErrorsFormSuccess(name, email))
	})

	// OOB warning — returns success content plus an OOB error banner.
	ar.e.GET(base+"/oob-warning", func(c echo.Context) error {
		requestID := middleware.GetRequestID(c)
		ec := hypermedia.ErrorContext{
			StatusCode: http.StatusOK,
			Message:    "Data loaded with warnings — some fields may be stale",
			Err:        errors.New("upstream cache returned partial data"),
			Route:      c.Request().URL.Path,
			RequestID:  requestID,
			Closable:   true,
			OOBTarget:  hypermedia.DefaultErrorStatusTarget,
			OOBSwap:    "innerHTML",
			Controls: []hypermedia.Control{
				hypermedia.DismissButton(hypermedia.LabelDismiss),
				hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID),
			},
		}
		return response.New(c).
			Component(views.ErrorsOOBSuccess()).
			OOBErrorStatus(ec).
			Send()
	})

	// Flaky endpoint — first call fails with retry button, second succeeds.
	ar.e.GET(base+"/flaky", func(c echo.Context) error {
		n := atomic.AddInt64(&flakyCount, 1)
		if n%2 == 1 {
			requestID := middleware.GetRequestID(c)
			c.Response().Status = http.StatusInternalServerError
			ec := hypermedia.ErrorContext{
				StatusCode: http.StatusInternalServerError,
				Message:    "Service temporarily unavailable",
				Err:        errors.New("connection to upstream timed out"),
				Route:      c.Request().URL.Path,
				RequestID:  requestID,
				Closable:   true,
				Controls: []hypermedia.Control{
					hypermedia.RetryButton("Retry", hypermedia.HxMethodGet,
						base+"/flaky", "#errors-retry-result").
						WithErrorTarget("#errors-retry-result"),
					hypermedia.ReportIssueButton(hypermedia.LabelReportIssue, requestID),
				},
			}
			return handler.RenderComponent(c, views.GalleryErrorPanel("flaky-error", ec))
		}
		return handler.RenderComponent(c, views.ErrorsFlakySuccess())
	})
}
