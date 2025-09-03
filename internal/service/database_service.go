package service

import (
	"narrabyte/internal/repository"
)

type DbSvc struct {
	User     UserService
	RepoLink RepoLinkService
}

func NewDbSvc(userRepo repository.UserRepository, repoLinkRepo repository.RepoLinkRepository) *DbSvc {
	return &DbSvc{
		User:     NewUserService(userRepo),
		RepoLink: NewRepoLinkService(repoLinkRepo),
	}
}
