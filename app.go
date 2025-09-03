package main

import (
	"context"
	"fmt"
	"narrabyte/internal/database"
	"narrabyte/internal/demo"
	"narrabyte/internal/repository"
	"narrabyte/internal/service"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm/logger"
)

// App struct
type App struct {
    ctx     context.Context
	DbSvc *service.DbSvc
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

	db, err := database.Init(database.Config{
		Path:     "narrabyte.db",
		LogLevel: logger.Info,
	})
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to open database: %v", err))
		return
	}

	userRepo := repository.NewUserRepository(db)
	repoLinkRepo := repository.NewRepoLinkRepository(db)
	a.DbSvc = service.NewDbSvc(userRepo, repoLinkRepo)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	if a.DbSvc.User != nil && name != "" {
		if _, err := a.DbSvc.User.Register(a.ctx, name); err != nil {
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
    ctx, cancel := context.WithCancel(a.ctx)
    a.demoCancel = cancel
    a.demoMu.Unlock()

    go func() {
        defer func() {
            a.demoMu.Lock()
            a.demoRunning = false
            a.demoCancel = nil
            a.demoMu.Unlock()
            // Notify frontend that the demo stream has finished
            runtime.EventsEmit(a.ctx, "demo:events:done")
        }()

        eventTypes := []string{"info", "debug", "warn", "error"}
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
                evt := demo.DemoEvent{
                    ID:        i,
                    Type:      eventTypes[(i-1)%len(eventTypes)],
                    Message:   fmt.Sprintf("Demo event #%d", i),
                    Timestamp: t,
                }
                runtime.EventsEmit(a.ctx, "demo:events", evt)
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
