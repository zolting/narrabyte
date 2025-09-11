package main

import (
	"context"
	"fmt"
	"narrabyte/internal/database"
	"narrabyte/internal/events"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm/logger"
)

// App struct
type App struct {
	ctx         context.Context
	Users       services.UserService
	RepoLinks   services.RepoLinkService
	AppSettings services.AppSettingsService
	dbClose     func() error
	demoMu      sync.Mutex
	demoRunning bool
	demoCancel  context.CancelFunc
	fumadocs    *services.FumadocsService
	git         *services.GitService
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

	// Wire services and inject only needed interfaces into App
	svc := services.NewDbServices(db)
	a.Users = svc.Users
	a.RepoLinks = svc.RepoLinks
	a.AppSettings = svc.AppSettings

	a.fumadocs = services.NewFumadocsService()
	a.git = services.NewGitService()

	// Capture DB close for graceful shutdown
	if sqlDB, err := db.DB(); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to get sql.DB: %v", err))
	} else {
		a.dbClose = sqlDB.Close
	}
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

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	if a.Users != nil && name != "" {
		if _, err := a.Users.Register(a.ctx, name); err != nil {
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
				evt := events.DemoEvent{
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

// LinkRepositories links the given repositories
func (a *App) LinkRepositories(projectName, docRepo, codebaseRepo string) error {
	if a.RepoLinks == nil {
		return fmt.Errorf("repo link service not available")
	}

	_, err := a.RepoLinks.Register(a.ctx, projectName, docRepo, codebaseRepo)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to link repositories: %v", err))
		return err
	}

	x, err := a.fumadocs.CreateFumadocsProject(docRepo)
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to create fumadocs project: %v", err))
		return fmt.Errorf("failed to create fumadocs project: %w", err)
	}
	runtime.LogInfo(a.ctx, x)

	runtime.LogInfo(a.ctx, fmt.Sprintf("Successfully linked project: %s, doc: %s with codebase: %s", projectName, docRepo, codebaseRepo))
	return nil
}

// GetAppSettings returns the current application settings
func (a *App) GetAppSettings() (*models.AppSettings, error) {
	if a.AppSettings == nil {
		return nil, fmt.Errorf("app settings service not available")
	}
	return a.AppSettings.Get(a.ctx)
}

// UpdateAppSettings updates theme and locale and returns the updated settings
func (a *App) UpdateAppSettings(theme, locale string) (*models.AppSettings, error) {
	if a.AppSettings == nil {
		return nil, fmt.Errorf("app settings service not available")
	}
	return a.AppSettings.Update(a.ctx, theme, locale)
}

// GetRepoLinks returns all repo links
func (a *App) GetRepoLinks() ([]models.RepoLink, error) {
	if a.RepoLinks == nil {
		return nil, fmt.Errorf("repo link service not available")
	}
	return a.RepoLinks.List(a.ctx, 100, 0)
}

// ListRepoBranches returns all branches of a repo
func (a *App) ListRepoBranches(repoPath string) ([]models.BranchInfo, error) {
	if a.git == nil {
		return nil, fmt.Errorf("git service not available")
	}
	return a.git.ListBranchesByPath(repoPath)
}
