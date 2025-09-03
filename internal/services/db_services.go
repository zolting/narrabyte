package services

import (
	"narrabyte/internal/repositories"

	"gorm.io/gorm"
)

// DbServices aggregates all domain services backed by the database.
// Fields use plural names (e.g., Users) to align with Go conventions
// seen in service/store containers.
type DbServices struct {
	Users     UserService
	RepoLinks RepoLinkService
}

// NewDbServices constructs the service container using repositories backed by db.
func NewDbServices(db *gorm.DB) *DbServices {
	userRepo := repositories.NewUserRepository(db)
	repoLinkRepo := repositories.NewRepoLinkRepository(db)

	return &DbServices{
		Users:     NewUserService(userRepo),
		RepoLinks: NewRepoLinkService(repoLinkRepo),
	}
}
