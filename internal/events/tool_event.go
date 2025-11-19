package events

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventInfo    EventType = "info"
	EventWarn    EventType = "warn"
	EventSuccess EventType = "success"
	EventError   EventType = "error"
)

const (
	LLMEventTool = "event:llm:tool"
	LLMGenerate  = "events:llm:generate"
	LLMEventDone = "events:llm:done"
)

// ToolEvent is a simple struct representing a backend event payload
type ToolEvent struct {
	ID         string            `json:"id"`
	Type       EventType         `json:"type"`
	Message    string            `json:"message"`
	Timestamp  time.Time         `json:"timestamp"`
	SessionKey string            `json:"sessionKey,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type contextKey string

const sessionContextKey contextKey = "narrabyte/events/session"

// WithSession returns a derived context annotated with the given session key
// so event emitters can automatically scope payloads.
func WithSession(ctx context.Context, sessionKey string) context.Context {
	if strings.TrimSpace(sessionKey) == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionContextKey, sessionKey)
}

// SessionFromContext extracts the session key associated with ctx.
func SessionFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(sessionContextKey).(string); ok {
		return v
	}
	return ""
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

// NewWarn creates a warn ToolEvent.
func NewWarn(message string) ToolEvent {
	return CreateToolEvent(EventWarn, message)
}

// NewError creates an error ToolEvent.
func NewError(message string) ToolEvent {
	return CreateToolEvent(EventError, message)
}

// NewSuccess creates a success ToolEvent.
func NewSuccess(message string) ToolEvent {
	return CreateToolEvent(EventSuccess, message)
}
