package services

import (
	"context"
	"errors"
	"time"

	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
)

type AppSettingsService interface {
	Get(ctx context.Context) (*models.AppSettings, error)
	Update(ctx context.Context, theme, locale string) (*models.AppSettings, error)
}

type appSettingsService struct {
	appSettings repositories.AppSettingsRepository
}

func NewAppSettingsService(appSettings repositories.AppSettingsRepository) AppSettingsService {
	return &appSettingsService{appSettings: appSettings}
}

func (s *appSettingsService) Get(ctx context.Context) (*models.AppSettings, error) {
	return s.appSettings.Get(ctx)
}

func (s *appSettingsService) Update(ctx context.Context, theme, locale string) (*models.AppSettings, error) {
	if theme == "" {
		return nil, errors.New("theme is required")
	}
	if locale == "" {
		return nil, errors.New("locale is required")
	}

	// Validate theme values
	if theme != "light" && theme != "dark" && theme != "system" {
		return nil, errors.New("theme must be 'light', 'dark', or 'system'")
	}

	// Get current settings
	current, err := s.appSettings.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	current.Theme = theme
	current.Locale = locale
	current.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.appSettings.Update(ctx, current); err != nil {
		return nil, err
	}

	return current, nil
}
