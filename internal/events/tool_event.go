package events

import "time"
import "github.com/google/uuid"

type EventType string

const (
	EventInfo  EventType = "info"
	EventDebug EventType = "debug"
	EventWarn  EventType = "warn"
	EventError EventType = "error"
)

const (
	EventToolStart    = "llm:tool:start"
	EventToolProgress = "llm:tool:progress"
	EventToolError    = "llm:tool:error"
	EventToolDone     = "llm:tool:done"
)

// ToolEvent is a simple struct representing a backend event payload
type ToolEvent struct {
	ID        uuid.UUID `json:"id"`
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func CreateToolEvent(eventType EventType, message string) ToolEvent {
	return ToolEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewInfo(message string) ToolEvent {
	return CreateToolEvent(EventInfo, message)
}

// NewDebug creates a debug ToolEvent.
func NewDebug(message string) ToolEvent {
	return CreateToolEvent(EventDebug, message)
}

// NewWarn creates a warn ToolEvent.
func NewWarn(message string) ToolEvent {
	return CreateToolEvent(EventWarn, message)
}

// NewError creates an error ToolEvent.
func NewError(message string) ToolEvent {
	return CreateToolEvent(EventError, message)
}
