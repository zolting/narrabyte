package mocks

import (
	"context"
	"narrabyte/internal/models"
)

type AppSettingsRepositoryMock struct {
	GetFunc    func(ctx context.Context) (*models.AppSettings, error)
	UpdateFunc func(ctx context.Context, settings *models.AppSettings) error
}

func (m *AppSettingsRepositoryMock) Get(ctx context.Context) (*models.AppSettings, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx)
	}
	return &models.AppSettings{
		ID:      1,
		Version: 1,
		Theme:   "system",
		Locale:  "en",
	}, nil
}

func (m *AppSettingsRepositoryMock) Update(ctx context.Context, settings *models.AppSettings) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, settings)
	}
	return nil
}
