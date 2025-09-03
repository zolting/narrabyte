package service

import (
	"context"
	"narrabyte/internal/models"
	"narrabyte/internal/repository"
)

type RepoLinkService interface {
	Register(ctx context.Context, documentationRepo, codebaseRepo string) (*models.RepoLink, error)
	Get(ctx context.Context, id uint) (*models.RepoLink, error)
	List(ctx context.Context, limit, offset int) ([]models.RepoLink, error)
}

type repoLinkService struct {
	repoLinks repository.RepoLinkRepository
}

func NewRepoLinkService(repoLinks repository.RepoLinkRepository) RepoLinkService {
	return &repoLinkService{repoLinks: repoLinks}
}

func (s *repoLinkService) Register(ctx context.Context, documentationRepo, codebaseRepo string) (*models.RepoLink, error) {
	link := &models.RepoLink{
		DocumentationRepo: documentationRepo,
		CodebaseRepo:      codebaseRepo,
	}
	if err := s.repoLinks.Create(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *repoLinkService) Get(ctx context.Context, id uint) (*models.RepoLink, error) {
	return s.repoLinks.FindByID(ctx, id)
}

func (s *repoLinkService) List(ctx context.Context, limit, offset int) ([]models.RepoLink, error) {
	return s.repoLinks.List(ctx, limit, offset)
}
