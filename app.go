package main

import (
	"context"
	"fmt"
	"narrabyte/internal/database"
	"narrabyte/internal/repository"
	"narrabyte/internal/service"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm/logger"
)

// App struct
type App struct {
	ctx   context.Context
	DbSvc *service.DbSvc
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
