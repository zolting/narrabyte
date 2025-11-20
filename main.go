package main

import (
	"context"
	"embed"
	"fmt"
	"narrabyte/internal/database"
	"narrabyte/internal/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"gorm.io/gorm/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	app := NewApp()

	db, err := database.Init(database.Config{
		LogLevel: logger.Info,
	})
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	if sqlDB, err := db.DB(); err == nil {
		app.dbClose = sqlDB.Close
	}

	//Create each service
	fumadocsService := services.NewFumadocsService()
	gitService := services.NewGitService()
	keyringService := services.NewKeyringService()
	dbService := services.NewDbServices(db, *fumadocsService, *gitService)
	clientService := services.NewClientService(dbService.RepoLinks, gitService, keyringService, dbService.GenerationSessions, dbService.ModelConfigs)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "Narrabyte",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "Narrabyte",
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			dbService.StartDbServices(ctx)
			fumadocsService.Startup(ctx)
			gitService.Startup(ctx)
			keyringService.Startup()

			//will have to check for this lowkey
			err := clientService.Startup(ctx)
			if err != nil {
				fmt.Println("Error starting client service:", err)
			}
		},
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
			dbService.RepoLinks,
			dbService.AppSettings,
			dbService.GenerationSessions,
			dbService.ModelConfigs,
			dbService.Templates,
			fumadocsService,
			gitService,
			clientService,
			keyringService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
