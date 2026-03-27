// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/internal/ssebroker"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

type kanbanRoutes struct {
	board  *demo.KanbanBoard
	actLog *demo.ActivityLog
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initKanbanRoutes(board *demo.KanbanBoard, actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	hypermedia.Link("/demo/kanban", "related", "/demo/approvals", "Approvals")
	hypermedia.Link("/demo/kanban", "related", "/demo/feed", "Feed")

	k := &kanbanRoutes{board: board, actLog: actLog, broker: broker}
	ar.e.GET("/demo/kanban", k.handleKanbanPage)
	ar.e.PATCH("/demo/kanban/tasks/:id/move", k.handleMoveTask)
}

func (k *kanbanRoutes) handleKanbanPage(c echo.Context) error {
	tasks := k.board.AllTasks()
	return handler.RenderBaseLayout(c, views.KanbanPage(tasks))
}

func (k *kanbanRoutes) handleMoveTask(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid task ID", err)
	}
	newStatus := c.QueryParam("status")
	task, ok := k.board.MoveTask(id, newStatus)
	if !ok {
		return handler.HandleHypermediaError(c, 404, "Task not found or invalid status", nil)
	}
	evt := k.actLog.Record("moved", "task", id, task.Title, "moved to "+newStatus)
	BroadcastActivity(k.broker, evt)
	tasks := k.board.AllTasks()
	return handler.RenderComponent(c, views.KanbanBoard(tasks))
}
