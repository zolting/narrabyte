package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"gorm.io/gorm/logger"
	"narrabyte/internal/database"
	"narrabyte/internal/services"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	// Create an instance of the app structure
	app := NewApp()

	//Initialize the database
	db, err := database.Init(database.Config{
		Path:     "narrabyte.db",
		LogLevel: logger.Info,
	})
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	//savais pas ou le mettre. Ici ou dans app.startup() ?
	if sqlDB, err := db.DB(); err == nil {
		app.dbClose = sqlDB.Close
	}

	//Create each service
	fumadocsService := services.NewFumadocsService()
	gitService := services.NewGitService()

	//Create repositories
	dbService := services.NewDbServices(db, *fumadocsService)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "narrabyte",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			dbService.StartDbServices(ctx)
			fumadocsService.Startup(ctx)
			gitService.Startup(ctx)
		},
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
			dbService.Users,
			dbService.RepoLinks,
			dbService.AppSettings,
			fumadocsService,
			gitService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
