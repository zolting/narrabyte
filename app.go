package main

import (
	"context"
	"fmt"
	"narrabyte/internal/events"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx         context.Context
	dbClose     func() error
	demoMu      sync.Mutex
	demoRunning bool
	demoCancel  context.CancelFunc
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is closing. Clean up resources here.
func (a *App) shutdown(ctx context.Context) {
	// Stop any running demo event stream
	a.demoMu.Lock()
	cancel := a.demoCancel
	a.demoMu.Unlock()
	if cancel != nil {
		cancel()
	}

	// Close database connection pool
	if a.dbClose != nil {
		if err := a.dbClose(); err != nil {
			runtime.LogError(ctx, fmt.Sprintf("failed to close database: %v", err))
		} else {
			runtime.LogInfo(ctx, "database closed")
		}
		a.dbClose = nil
	}
}

// SelectDirectory opens a native directory picker dialog
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// StartDemoEvents starts emitting demo events periodically to the frontend via Wails events
// It will no-op if a demo events stream is already running
func (a *App) StartDemoEvents() {
	a.demoMu.Lock()
	if a.demoRunning {
		// already running; ignore duplicate starts
		a.demoMu.Unlock()
		return
	}
	a.demoRunning = true
	ctx, cancel := context.WithCancel(a.ctx)
	a.demoCancel = cancel
	a.demoMu.Unlock()

	go func() {
		defer func() {
			a.demoMu.Lock()
			a.demoRunning = false
			a.demoCancel = nil
			a.demoMu.Unlock()
			// Notify frontend that the demo events stream has finished
			runtime.EventsEmit(a.ctx, "events:demo:done")
		}()

		eventTypes := []events.EventType{events.EventInfo, events.EventDebug, events.EventWarn, events.EventError}
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case t := <-ticker.C:
				i++
				if i > 15 {
					return
				}
				evt := events.ToolEvent{
					ID:        i,
					Type:      eventTypes[(i-1)%len(eventTypes)],
					Message:   fmt.Sprintf("Demo event #%d", i),
					Timestamp: t,
				}
				runtime.EventsEmit(a.ctx, "events:demo", evt)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopDemoEvents cancels the running demo event stream, if any
func (a *App) StopDemoEvents() {
	a.demoMu.Lock()
	cancel := a.demoCancel
	running := a.demoRunning
	a.demoMu.Unlock()
	if running && cancel != nil {
		cancel()
	}
}
