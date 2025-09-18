package services

import (
	"context"
	"errors"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"narrabyte/internal/events"
	"sync"
)

type EventEmitterService struct {
	context context.Context
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

func (e *EventEmitterService) Startup(ctx context.Context) {
	e.context = ctx
}

func NewEventEmitterService() *EventEmitterService {
	return &EventEmitterService{}
}

func (e *EventEmitterService) StartStream() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.context == nil || e.running {
		return false
	}

	e.context, e.cancel = context.WithCancel(e.context)
	e.running = true
	runtime.EventsEmit(e.context, "Started event stream", nil)
	return true
}

func (e *EventEmitterService) EmitEvent(name string, event events.ToolEvent) error {
	if e == nil || e.context == nil {
		return errors.New("The context and emitter are not initialized")
	}

	if e.running {
		runtime.EventsEmit(e.context, name, event)
	}
	return nil
}

func (e *EventEmitterService) StopStream() {
	e.mu.Lock()
	defer e.mu.Unlock()
	cancel := e.cancel
	running := e.running
	if running && cancel != nil {
		cancel()
	}
}
