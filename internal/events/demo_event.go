package events

import "time"

type EventType string

const (
	EventInfo  EventType = "info"
	EventDebug EventType = "debug"
	EventWarn  EventType = "warn"
	EventError EventType = "error"
)

// DemoEvent is a simple struct representing a backend event payload
type DemoEvent struct {
	ID        int       `json:"id"`
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
