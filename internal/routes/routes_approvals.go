// setup:feature:demo

package routes

import (
	"fmt"

	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type approvalRoutes struct {
	queue  *demo.ApprovalQueue
	actLog *demo.ActivityLog
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initApprovalRoutes(queue *demo.ApprovalQueue, actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	hypermedia.Link("/demo/approvals", "related", "/demo/feed", "Feed")

	a := &approvalRoutes{queue: queue, actLog: actLog, broker: broker}
	ar.e.GET("/demo/approvals", a.handleApprovalsPage)
	ar.e.POST("/demo/approvals/:id/:action", a.handleApprovalAction)
}

func (a *approvalRoutes) handleApprovalsPage(c echo.Context) error {
	requests := a.queue.AllRequests()
	return handler.RenderBaseLayout(c, views.ApprovalsPage(requests))
}

func (a *approvalRoutes) handleApprovalAction(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid request ID", err)
	}
	action := c.Param("action")
	req, ok := a.queue.TransitionRequest(id, action, "Admin")
	if !ok {
		return handler.HandleHypermediaError(c, 400, fmt.Sprintf("Cannot %s request %d", action, id), nil)
	}
	evt := a.actLog.Record(action, "approval", id, req.Title, fmt.Sprintf("$%.2f %s", req.Amount, action))
	BroadcastActivity(a.broker, evt)
	return handler.RenderComponent(c, views.ApprovalCard(req))
}
