package services

import (
	"context"
	"narrabyte/internal/repositories"

	"gorm.io/gorm"
)

// DbServices aggregates all domain services backed by the database.
// Fields use plural names (e.g., Users) to align with Go conventions
// seen in service/store containers.
type DbServices struct {
	RepoLinks          RepoLinkService
	AppSettings        AppSettingsService
	GenerationSessions GenerationSessionService
}

// NewDbServices constructs the service container using repositories backed by db.
func NewDbServices(db *gorm.DB, fumaDocService FumadocsService, gitService GitService) *DbServices {
	repoLinkRepo := repositories.NewRepoLinkRepository(db)
	appSettingsRepo := repositories.NewAppSettingsRepository(db)
	genSessionRepo := repositories.NewGenerationSessionRepository(db)

	return &DbServices{
		RepoLinks:          NewRepoLinkService(repoLinkRepo, fumaDocService, gitService),
		AppSettings:        NewAppSettingsService(appSettingsRepo),
		GenerationSessions: NewGenerationSessionService(genSessionRepo),
	}
}

func (db *DbServices) StartDbServices(ctx context.Context) {
	db.RepoLinks.Startup(ctx)
	db.AppSettings.Startup(ctx)
	db.GenerationSessions.Startup(ctx)
}
