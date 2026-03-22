// setup:feature:demo

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initReportDemoRoutes() {
	// Override the report-issue POST to show a simulated email modal instead of
	// the default browser alert. The modal displays what an IssueReporter
	// implementation would send to a support team.
	ar.e.POST("/report-issue/:requestID", func(c echo.Context) error {
		requestID := c.Param("requestID")
		description := c.FormValue("description")

		var trace *views.ReportEmailData
		if ar.reqLogStore != nil && requestID != "" {
			t, _ := ar.reqLogStore.Get(c.Request().Context(), requestID)
			if t != nil {
				jsonBytes, _ := json.MarshalIndent(map[string]any{
					"request_id":  t.RequestID,
					"error_chain": t.ErrorChain,
					"status_code": t.StatusCode,
					"route":       t.Route,
					"method":      t.Method,
					"user_agent":  t.UserAgent,
					"remote_ip":   t.RemoteIP,
					"user_id":     t.UserID,
					"created_at":  t.CreatedAt,
					"log_entries": t.Entries,
				}, "", "  ")
				trace = &views.ReportEmailData{
					To:          "support@example.com",
					From:        t.UserID,
					Subject:     fmt.Sprintf("[Error %d] %s %s", t.StatusCode, t.Method, t.Route),
					Description: description,
					RequestID:   requestID,
					Attachment:  string(jsonBytes),
				}
			}
		}
		if trace == nil {
			trace = &views.ReportEmailData{
				To:          "support@example.com",
				Subject:     fmt.Sprintf("[Error] Request %s", requestID),
				Description: description,
				RequestID:   requestID,
				Attachment:  "{}",
			}
		}

		c.Response().Header().Set("HX-Trigger", `{"showReportModal":""}`)
		c.Response().Status = http.StatusOK
		return handler.RenderComponent(c, views.ReportEmailModal(*trace))
	})
}
