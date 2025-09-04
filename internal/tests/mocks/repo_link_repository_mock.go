package mocks

import (
	"context"
	"narrabyte/internal/models"
)

type RepoLinkRepositoryMock struct {
	CreateFunc   func(ctx context.Context, link *models.RepoLink) error
	FindByIDFunc func(ctx context.Context, id uint) (*models.RepoLink, error)
	ListFunc     func(ctx context.Context, limit, offset int) ([]models.RepoLink, error)
}

func (m *RepoLinkRepositoryMock) Create(ctx context.Context, link *models.RepoLink) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, link)
	}
	return nil
}

func (m *RepoLinkRepositoryMock) FindByID(ctx context.Context, id uint) (*models.RepoLink, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *RepoLinkRepositoryMock) List(ctx context.Context, limit, offset int) ([]models.RepoLink, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset)
	}
	return []models.RepoLink{}, nil
}
