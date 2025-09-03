package services

import (
	"narrabyte/internal/repositories"

	"gorm.io/gorm"
)

type DbSvc struct {
	User     UserService
	RepoLink RepoLinkService
}

func NewDbSvc(db *gorm.DB) *DbSvc {
	userRepo := repositories.NewUserRepository(db)
	repoLinkRepo := repositories.NewRepoLinkRepository(db)

	return &DbSvc{
		User:     NewUserService(userRepo),
		RepoLink: NewRepoLinkService(repoLinkRepo),
	}
}
