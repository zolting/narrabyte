package main

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"narrabyte/internal/events"
	"sync"
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
	events.EnableRuntimeEmitter()
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
