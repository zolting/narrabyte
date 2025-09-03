package services

import (
	"gorm.io/gorm"
	"narrabyte/internal/repositories"
)

// Services aggregates all domain services backed by the database.
// Fields use plural names (e.g., Users) to align with Go conventions
// seen in service/store containers.
type Services struct {
	Users     UserService
	RepoLinks RepoLinkService
}

// NewServices constructs the service container using repositories backed by db.
func NewServices(db *gorm.DB) *Services {
	userRepo := repositories.NewUserRepository(db)
	repoLinkRepo := repositories.NewRepoLinkRepository(db)

	return &Services{
		Users:     NewUserService(userRepo),
		RepoLinks: NewRepoLinkService(repoLinkRepo),
	}
}
