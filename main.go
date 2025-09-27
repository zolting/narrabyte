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
	"gorm.io/gorm/logger"
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
	dbService := services.NewDbServices(db, *fumadocsService, *gitService)
	clientService := services.NewClientService(dbService.RepoLinks, gitService)

	//Create the keyring (à valider!!!!)
	keyringService := services.NewKeyringService()

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
			//Pas besoin du context pour le keyring, propre a l'OS de l'utilisateur
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
