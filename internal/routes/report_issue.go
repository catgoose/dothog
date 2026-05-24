package routes

import (
	"encoding/json"

	// setup:feature:demo:start
	"catgoose/dothog/internal/demo"
	// setup:feature:demo:end
	"catgoose/dothog/internal/htmxutil"
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"github.com/catgoose/linkwell"
	"net/http"

	corecomponents "catgoose/dothog/web/components/core"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
)

func (ar *AppRoutes) initReportIssueRoutes() {
	// POST /report-issue[/:requestID] — accepts a report, passes log entries
	// to the configured IssueReporter, and triggers a browser alert.
	reportHandler := func(c echo.Context) error {
		requestID := c.Param("requestID")
		description := c.FormValue("description")
		var trace *promolog.Trace
		if ar.deps.ReqLogStore != nil && requestID != "" {
			var err error
			trace, err = ar.deps.ReqLogStore.Get(c.Request().Context(), requestID)
			if err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to retrieve error trace for report",
					"request_id", requestID, "error", err)
			}
		}
		if err := ar.deps.IssueReporter.Report(requestID, description, trace); err != nil {
			logger.WithContext(c.Request().Context()).Error("Issue report failed",
				"reported_request_id", requestID, "error", err)
			_ = htmxutil.New().
				TriggerDetail("showAlert", "Failed to submit report. Please try again.").
				ReswapNone().
				Write(c.Response())
			return c.String(http.StatusInternalServerError, "")
		}
		// setup:feature:demo:start
		ar.persistReportToDemoDB(c, requestID, description, trace)
		// setup:feature:demo:end
		_ = htmxutil.New().
			TriggerDetail("showAlert", "Issue reported. Thank you for your feedback!").
			ReswapNone().
			Write(c.Response())
		return c.String(http.StatusOK, "")
	}
	reports := ar.e.Group("/report-issue")
	reports.POST("", reportHandler)
	reports.POST("/:requestID", reportHandler)

	// GET /report-issue/:requestID — returns the Report Issue modal fragment.
	// The modal auto-opens via HyperScript on load.
	reports.GET("/:requestID", func(c echo.Context) error {
		requestID := c.Param("requestID")
		cfg := linkwell.ReportIssueModal(requestID)
		return handler.RenderComponent(c, corecomponents.ReportIssueModal(cfg))
	})
}

// setup:feature:demo:start

// persistReportToDemoDB writes the submitted issue into the demo SQLite
// store so the admin error-reports page can list it. Demo-feature only:
// no-op when ar.demoDB is nil, since main.go warns and continues when
// db/demo.db is unavailable.
func (ar *AppRoutes) persistReportToDemoDB(c echo.Context, requestID, description string, trace *promolog.Trace) {
	if ar.demoDB == nil {
		return
	}
	logEntries := "[]"
	if trace != nil {
		if b, err := json.Marshal(trace.Entries); err == nil {
			logEntries = string(b)
		}
	}
	var statusCode int
	var route string
	if trace != nil {
		statusCode = trace.StatusCode
		route = trace.Route
	}
	report := demo.ErrorReport{
		RequestID:   requestID,
		Description: description,
		Route:       route,
		StatusCode:  statusCode,
		UserAgent:   c.Request().UserAgent(),
		LogEntries:  logEntries,
	}
	if _, err := ar.demoDB.InsertErrorReport(c.Request().Context(), report); err != nil {
		logger.WithContext(c.Request().Context()).Error("Failed to store error report",
			"request_id", requestID, "error", err)
	}
}

// setup:feature:demo:end
