package mocks

import (
	"context"
	"narrabyte/internal/models"
)

type UserRepositoryMock struct {
	CreateFunc   func(ctx context.Context, u *models.User) error
	FindByIDFunc func(ctx context.Context, id uint) (*models.User, error)
	ListFunc     func(ctx context.Context, limit, offset int) ([]models.User, error)
	UpdateFunc   func(ctx context.Context, u *models.User) error
	DeleteFunc   func(ctx context.Context, id uint) error
}

func (m *UserRepositoryMock) Create(ctx context.Context, u *models.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return nil
}

func (m *UserRepositoryMock) FindByID(ctx context.Context, id uint) (*models.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *UserRepositoryMock) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset)
	}
	return []models.User{}, nil
}

func (m *UserRepositoryMock) Update(ctx context.Context, u *models.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, u)
	}
	return nil
}

func (m *UserRepositoryMock) Delete(ctx context.Context, id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
