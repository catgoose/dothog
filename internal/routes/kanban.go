// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"
	"github.com/catgoose/tavern"
	"net/http"

	"github.com/labstack/echo/v4"
)

type kanbanRoutes struct {
	board  *demo.KanbanBoard
	actLog *demo.ActivityLog
	broker *tavern.SSEBroker
}

func (ar *AppRoutes) initKanbanRoutes(board *demo.KanbanBoard, actLog *demo.ActivityLog, broker *tavern.SSEBroker) {
	k := &kanbanRoutes{board: board, actLog: actLog, broker: broker}
	ar.e.GET("/apps/kanban", k.handleKanbanPage)
	ar.e.PATCH("/apps/kanban/tasks/:id", k.handleMoveTask)
}

func (k *kanbanRoutes) handleKanbanPage(c echo.Context) error {
	tasks := k.board.AllTasks()
	return handler.RenderBaseLayout(c, views.KanbanPage(tasks))
}

func (k *kanbanRoutes) handleMoveTask(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid task ID", err)
	}
	newStatus := c.FormValue("status")
	task, ok := k.board.MoveTask(id, newStatus)
	if !ok {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Task not found or invalid status", nil)
	}
	evt := k.actLog.Record("moved", "task", id, task.Title, "moved to "+newStatus)
	BroadcastActivity(k.broker, evt)
	tasks := k.board.AllTasks()
	return handler.RenderComponent(c, views.KanbanBoard(tasks))
}
