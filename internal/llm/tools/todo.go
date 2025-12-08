package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TodoStatus represents the state of a todo item
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// Todo represents a single task item
type Todo struct {
	Content    string     `json:"content" jsonschema:"required,description=The task description (imperative form: 'Generate API documentation')"`
	ActiveForm string     `json:"activeForm" jsonschema:"required,description=Present continuous form shown during execution ('Generating API documentation')"`
	Status     TodoStatus `json:"status" jsonschema:"required,enum=pending|in_progress|completed|cancelled,description=Current status of the task"`
}

// TodoSession manages todos for a single session
type TodoSession struct {
	SessionID string
	Todos     []Todo
	UpdatedAt time.Time
	mu        sync.RWMutex
}

// Global session storage
var (
	todoSessions   = make(map[string]*TodoSession)
	todoSessionsMu sync.RWMutex
)

// GetTodoSession retrieves or creates a todo session
func GetTodoSession(sessionID string) *TodoSession {
	todoSessionsMu.Lock()
	defer todoSessionsMu.Unlock()

	if session, exists := todoSessions[sessionID]; exists {
		return session
	}

	session := &TodoSession{
		SessionID: sessionID,
		Todos:     []Todo{},
		UpdatedAt: time.Now(),
	}
	todoSessions[sessionID] = session
	return session
}

// UpdateTodos performs a FULL REPLACEMENT of the todo list for a session (any existing todos not in the new list are deleted)
func (s *TodoSession) UpdateTodos(todos []Todo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Todos = todos
	s.UpdatedAt = time.Now()
}

// GetTodos returns a copy of the current todos
func (s *TodoSession) GetTodos() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	todos := make([]Todo, len(s.Todos))
	copy(todos, s.Todos)
	return todos
}

// ClearTodoSession removes a session's todos
func ClearTodoSession(sessionID string) {
	todoSessionsMu.Lock()
	defer todoSessionsMu.Unlock()
	delete(todoSessions, sessionID)
}

// TodoWriteInput defines the input structure for updating todos
type TodoWriteInput struct {
	Todos []Todo `json:"todos" jsonschema:"required,description=The COMPLETE todo list (replaces existing list entirely - must include all tasks you want to keep)"`
}

// TodoWriteOutput defines the output structure after updating todos
type TodoWriteOutput struct {
	Title    string         `json:"title"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TodoReadInput defines the input structure for reading todos
// This struct has no fields since the tool requires no input parameters
type TodoReadInput struct{}

// UnmarshalJSON implements custom JSON unmarshaling to handle empty or missing input
func (t *TodoReadInput) UnmarshalJSON(data []byte) error {
	// Accept empty input - this tool requires no parameters
	if len(data) == 0 {
		return nil
	}

	// Trim whitespace
	trimmed := string(data)
	trimmed = strings.TrimSpace(trimmed)

	// Accept empty string or empty object
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return nil
	}

	// If any other JSON is provided, try to parse it as an empty object to validate it
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Accept any valid JSON object even if it has fields (we'll just ignore them)
	return nil
}

// TodoReadOutput defines the output structure for reading todos
type TodoReadOutput struct {
	Title    string         `json:"title"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// WriteTodo replaces the entire todo list for the current session (full replacement operation)
func WriteTodo(ctx context.Context, in *TodoWriteInput) (*TodoWriteOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("todo input is required")
	}
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return nil, fmt.Errorf("no session ID found in context")
	}

	session := GetTodoSession(sessionID)

	normalizedTodos := make([]Todo, 0, len(in.Todos))
	for _, todo := range in.Todos {
		status := todo.Status
		switch todo.Status {
		case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
		default:
			status = TodoStatusPending
		}
		normalizedTodos = append(normalizedTodos, Todo{
			Content:    strings.TrimSpace(todo.Content),
			ActiveForm: strings.TrimSpace(todo.ActiveForm),
			Status:     status,
		})
	}

	session.UpdateTodos(normalizedTodos)

	// Calculate pending count
	pendingCount := 0
	inProgressCount := 0
	for _, todo := range normalizedTodos {
		if todo.Status == TodoStatusPending {
			pendingCount++
		} else if todo.Status == TodoStatusInProgress {
			inProgressCount++
		}
	}

	// Serialize todos for output
	todosJSON, err := json.MarshalIndent(normalizedTodos, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize todos: %w", err)
	}

	title := fmt.Sprintf("%d pending, %d in progress", pendingCount, inProgressCount)

	return &TodoWriteOutput{
		Title:  title,
		Output: string(todosJSON),
		Metadata: map[string]any{
			"todos":           normalizedTodos,
			"sessionID":       sessionID,
			"pendingCount":    pendingCount,
			"inProgressCount": inProgressCount,
			"totalCount":      len(normalizedTodos),
		},
	}, nil
}

// ReadTodo retrieves the current todo list for the session
func ReadTodo(ctx context.Context, in *TodoReadInput) (*TodoReadOutput, error) {
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		return nil, fmt.Errorf("no session ID found in context")
	}

	session := GetTodoSession(sessionID)
	todos := session.GetTodos()

	// Calculate counts
	pendingCount := 0
	inProgressCount := 0
	for _, todo := range todos {
		if todo.Status == TodoStatusPending {
			pendingCount++
		} else if todo.Status == TodoStatusInProgress {
			inProgressCount++
		}
	}

	// Serialize todos for output
	todosJSON, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize todos: %w", err)
	}

	title := fmt.Sprintf("%d pending, %d in progress", pendingCount, inProgressCount)
	if len(todos) == 0 {
		title = "No todos"
	}

	return &TodoReadOutput{
		Title:  title,
		Output: string(todosJSON),
		Metadata: map[string]any{
			"todos":           todos,
			"sessionID":       sessionID,
			"pendingCount":    pendingCount,
			"inProgressCount": inProgressCount,
			"totalCount":      len(todos),
		},
	}, nil
}
