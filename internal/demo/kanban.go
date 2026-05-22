// setup:feature:demo

package demo

import "sync"

// KanbanStatuses enumerates board columns left-to-right.
var KanbanStatuses = []string{"backlog", "todo", "in_progress", "review", "done"}

// KanbanPriorities enumerates priority levels in ascending urgency.
var KanbanPriorities = []string{"low", "medium", "high", "critical"}

// KanbanTask is one card on the board; Status must be one of KanbanStatuses.
type KanbanTask struct {
	Title       string
	Description string
	Status      string
	Priority    string
	Assignee    string
	ID          int
}

// KanbanBoard is a thread-safe in-memory kanban board.
type KanbanBoard struct {
	tasks  []KanbanTask
	nextID int
	mu     sync.RWMutex
}

// NewKanbanBoard pre-seeds 9 sample cards spread across all five status columns.
func NewKanbanBoard() *KanbanBoard {
	b := &KanbanBoard{}
	seed := []KanbanTask{
		{Title: "Set up CI pipeline", Description: "Configure GitHub Actions for automated builds and tests", Status: "done", Priority: "high", Assignee: "James S."},
		{Title: "Design database schema", Description: "Create ERD and define tables for the core domain", Status: "done", Priority: "high", Assignee: "Mary J."},
		{Title: "Implement user authentication", Description: "Add login, registration, and session management", Status: "review", Priority: "critical", Assignee: "James S."},
		{Title: "Build dashboard UI", Description: "Create the main dashboard layout with summary cards", Status: "in_progress", Priority: "high", Assignee: "Mary J."},
		{Title: "Add search functionality", Description: "Full-text search across tasks and people", Status: "in_progress", Priority: "medium", Assignee: "Robert W."},
		{Title: "Write API documentation", Description: "Document all REST endpoints with examples", Status: "todo", Priority: "medium", Assignee: "Patricia B."},
		{Title: "Set up error monitoring", Description: "Integrate Sentry for production error tracking", Status: "todo", Priority: "low", Assignee: "James S."},
		{Title: "Performance testing", Description: "Load test critical endpoints and optimize slow queries", Status: "backlog", Priority: "medium", Assignee: "Mary J."},
		{Title: "Mobile responsive design", Description: "Ensure all pages work well on mobile devices", Status: "backlog", Priority: "low", Assignee: "Robert W."},
	}
	for _, t := range seed {
		b.nextID++
		t.ID = b.nextID
		b.tasks = append(b.tasks, t)
	}
	return b
}

// AllTasks is a defensive copy; safe to mutate independently of the board.
func (b *KanbanBoard) AllTasks() []KanbanTask {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]KanbanTask, len(b.tasks))
	copy(out, b.tasks)
	return out
}

// TasksByStatus filters by column, preserving insertion order.
func (b *KanbanBoard) TasksByStatus(status string) []KanbanTask {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []KanbanTask
	for _, t := range b.tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// GetTask looks up by ID; ok is false when the task is not found.
func (b *KanbanBoard) GetTask(id int) (KanbanTask, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, t := range b.tasks {
		if t.ID == id {
			return t, true
		}
	}
	return KanbanTask{}, false
}

// MoveTask reassigns Status only; ok is false if id is unknown.
func (b *KanbanBoard) MoveTask(id int, newStatus string) (KanbanTask, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.tasks {
		if b.tasks[i].ID == id {
			b.tasks[i].Status = newStatus
			return b.tasks[i], true
		}
	}
	return KanbanTask{}, false
}

// UpdateTask overwrites the card's editable fields in-place; Status is preserved (use MoveTask to change it).
func (b *KanbanBoard) UpdateTask(id int, title, description, priority, assignee string) (KanbanTask, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.tasks {
		if b.tasks[i].ID == id {
			b.tasks[i].Title = title
			b.tasks[i].Description = description
			b.tasks[i].Priority = priority
			b.tasks[i].Assignee = assignee
			return b.tasks[i], true
		}
	}
	return KanbanTask{}, false
}
