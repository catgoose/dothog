package routes

import (
	"catgoose/dothog/internal/logger"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"net/http"

	corecomponents "catgoose/dothog/web/components/core"

	"github.com/catgoose/promolog"
	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initReportIssueRoutes() {
	// POST /report-issue[/:requestID] — accepts a report, passes log entries
	// to the configured IssueReporter, and triggers a browser alert.
	reportHandler := func(c echo.Context) error {
		requestID := c.Param("requestID")
		description := c.FormValue("description")
		var trace *promolog.ErrorTrace
		if ar.reqLogStore != nil && requestID != "" {
			var err error
			trace, err = ar.reqLogStore.Get(c.Request().Context(), requestID)
			if err != nil {
				logger.WithContext(c.Request().Context()).Error("Failed to retrieve error trace for report",
					"request_id", requestID, "error", err)
			}
		}
		if err := ar.issueReporter.Report(requestID, description, trace); err != nil {
			logger.WithContext(c.Request().Context()).Error("Issue report failed",
				"reported_request_id", requestID, "error", err)
			c.Response().Header().Set("HX-Trigger", `{"showAlert":"Failed to submit report. Please try again."}`)
			c.Response().Header().Set("HX-Reswap", "none")
			return c.String(http.StatusInternalServerError, "")
		}
		c.Response().Header().Set("HX-Trigger", `{"showAlert":"Issue reported. Thank you for your feedback!"}`)
		c.Response().Header().Set("HX-Reswap", "none")
		return c.String(http.StatusOK, "")
	}
	ar.e.POST("/report-issue", reportHandler)
	ar.e.POST("/report-issue/:requestID", reportHandler)

	// GET /report-issue/:requestID — returns the Report Issue modal fragment.
	// The modal auto-opens via HyperScript on load.
	ar.e.GET("/report-issue/:requestID", func(c echo.Context) error {
		requestID := c.Param("requestID")
		cfg := hypermedia.ReportIssueModal(requestID)
		return handler.RenderComponent(c, corecomponents.ReportIssueModal(cfg))
	})
}
