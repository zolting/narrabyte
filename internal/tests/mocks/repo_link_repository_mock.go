package mocks

import (
	"context"
	"narrabyte/internal/models"
)

type RepoLinkRepositoryMock struct {
	CreateFunc       func(ctx context.Context, link *models.RepoLink) error
	FindByIDFunc     func(ctx context.Context, id uint) (*models.RepoLink, error)
	ListFunc         func(ctx context.Context, limit, offset int) ([]models.RepoLink, error)
	UpdateFunc       func(ctx context.Context, link *models.RepoLink) error
	DeleteFunc       func(ctx context.Context, id uint) error
	UpdateOrderFunc  func(ctx context.Context, updates []models.RepoLinkOrderUpdate) error
	IncrementAllFunc func(ctx context.Context) error
	GetMaxIndexFunc  func(ctx context.Context) (int, error)
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

func (m *RepoLinkRepositoryMock) Update(ctx context.Context, link *models.RepoLink) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, link)
	}
	return nil
}

func (m *RepoLinkRepositoryMock) Delete(ctx context.Context, id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *RepoLinkRepositoryMock) UpdateOrder(ctx context.Context, updates []models.RepoLinkOrderUpdate) error {
	if m.UpdateOrderFunc != nil {
		return m.UpdateOrderFunc(ctx, updates)
	}
	return nil
}

func (m *RepoLinkRepositoryMock) IncrementAllIndexes(ctx context.Context) error {
	if m.IncrementAllFunc != nil {
		return m.IncrementAllFunc(ctx)
	}
	return nil
}

func (m *RepoLinkRepositoryMock) GetMaxIndex(ctx context.Context) (int, error) {
	if m.GetMaxIndexFunc != nil {
		return m.GetMaxIndexFunc(ctx)
	}
	return 0, nil
}
