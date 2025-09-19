package events

import (
	"github.com/google/uuid"
	"time"
)

type EventType string

const (
	EventInfo  EventType = "info"
	EventDebug EventType = "debug"
	EventWarn  EventType = "warn"
	EventError EventType = "error"
)

const (
	LLMEventTool = "event:llm:tool"
	LLMEventDone = "events:llm:done"
)

// ToolEvent is a simple struct representing a backend event payload
type ToolEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func CreateToolEvent(eventType EventType, message string) ToolEvent {
	return ToolEvent{
		ID:        uuid.NewString(),
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewInfo creates an info ToolEvent.
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
