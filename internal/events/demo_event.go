package events

import "time"

// DemoEvent is a simple struct representing a backend event payload
type DemoEvent struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
