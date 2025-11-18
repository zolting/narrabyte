package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	LLMEventTodo = "event:llm:todo"
)

// TodoItem represents a single task in the todo list
type TodoItem struct {
	Content    string `json:"content"`
	ActiveForm string `json:"activeForm"`
	Status     string `json:"status"`
}

// TodoEvent represents a todo list update event
type TodoEvent struct {
	ID         string     `json:"id"`
	Todos      []TodoItem `json:"todos"`
	Timestamp  time.Time  `json:"timestamp"`
	SessionKey string     `json:"sessionKey,omitempty"`
}

// EmitTodoUpdate emits a todo update event to the frontend
// The todos parameter should be []TodoItem
func EmitTodoUpdate(ctx context.Context, todos []TodoItem) {
	sessionKey := SessionFromContext(ctx)

	evt := TodoEvent{
		ID:         uuid.NewString(),
		Todos:      todos,
		Timestamp:  time.Now(),
		SessionKey: sessionKey,
	}

	runtime.EventsEmit(ctx, LLMEventTodo, evt)
}
