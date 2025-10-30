package mocks

import (
	"context"
	"narrabyte/internal/models"
)

type TemplateRepositoryMock struct {
	GetFunc    func(ctx context.Context, id uint) (*models.Template, error)
	GetAllFunc func(ctx context.Context) ([]*models.Template, error)
	CreateFunc func(ctx context.Context, template *models.Template) error
	UpdateFunc func(ctx context.Context, template *models.Template) error
	DeleteFunc func(ctx context.Context, id uint) error
}

func (m *TemplateRepositoryMock) Get(ctx context.Context, id uint) (*models.Template, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *TemplateRepositoryMock) GetAll(ctx context.Context) ([]*models.Template, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	return []*models.Template{}, nil
}

func (m *TemplateRepositoryMock) Create(ctx context.Context, template *models.Template) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, template)
	}
	return nil
}

func (m *TemplateRepositoryMock) Update(ctx context.Context, template *models.Template) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, template)
	}
	return nil
}

func (m *TemplateRepositoryMock) Delete(ctx context.Context, id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
