package main

import (
	"context"
	"fmt"
	"narrabyte/internal/database"
	"narrabyte/internal/repository"
	"narrabyte/internal/service"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm/logger"

	"gorm.io/gorm"
)

// App struct
type App struct {
    ctx     context.Context
    DB      *gorm.DB
    UserSvc service.UserService
    demoMu      sync.Mutex
    demoRunning bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	db, err := database.Init(database.Config{
		Path:     "narrabyte.db",
		LogLevel: logger.Info,
	})
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to open database: %v", err))
		return
	}
	a.DB = db

	userRepo := repository.NewUserRepository(a.DB)
	a.UserSvc = service.NewUserService(userRepo)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	if a.UserSvc != nil && name != "" {
		if _, err := a.UserSvc.Register(a.ctx, name); err != nil {
			runtime.LogError(a.ctx, fmt.Sprintf("failed to create user: %v", err))
		}
	}

	return fmt.Sprintf("Hello bruvddd %s, It's show time!", name)
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

// DemoEvent is a simple struct representing a backend event payload
type DemoEvent struct {
    ID        int       `json:"id"`
    Type      string    `json:"type"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}

// StartDemoEvents starts emitting demo events periodically to the frontend via Wails events
// It will no-op if a demo stream is already running
func (a *App) StartDemoEvents() {
    a.demoMu.Lock()
    if a.demoRunning {
        // already running; ignore duplicate starts
        a.demoMu.Unlock()
        return
    }
    a.demoRunning = true
    a.demoMu.Unlock()

    go func() {
        defer func() {
            a.demoMu.Lock()
            a.demoRunning = false
            a.demoMu.Unlock()
            // Notify frontend that the demo stream has finished
            runtime.EventsEmit(a.ctx, "demo:events:done")
        }()

        eventTypes := []string{"info", "debug", "warn", "error"}
        for i := 1; i <= 15; i++ {
            evt := DemoEvent{
                ID:        i,
                Type:      eventTypes[(i-1)%len(eventTypes)],
                Message:   fmt.Sprintf("Demo event #%d", i),
                Timestamp: time.Now(),
            }
            runtime.EventsEmit(a.ctx, "demo:events", evt)
            time.Sleep(1 * time.Second)
        }
    }()
}
