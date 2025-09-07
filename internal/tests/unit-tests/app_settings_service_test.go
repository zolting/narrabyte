package unit_tests

import (
	"context"
	"errors"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"narrabyte/internal/utils"
	"testing"
)

func TestAppSettingsService_Get_Success(t *testing.T) {
	expectedSettings := &models.AppSettings{
		ID:      1,
		Version: 1,
		Theme:   "dark",
		Locale:  "fr",
	}

	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return expectedSettings, nil
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	settings, err := service.Get(ctx)
	utils.NilError(t, err)
	utils.Equal(t, settings.ID, expectedSettings.ID)
	utils.Equal(t, settings.Version, expectedSettings.Version)
	utils.Equal(t, settings.Theme, expectedSettings.Theme)
	utils.Equal(t, settings.Locale, expectedSettings.Locale)
}

func TestAppSettingsService_Get_RepositoryError(t *testing.T) {
	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return nil, errors.New("database error")
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Get(ctx)
	utils.Equal(t, err.Error(), "database error")
}

func TestAppSettingsService_Update_Success(t *testing.T) {
	currentSettings := &models.AppSettings{
		ID:      1,
		Version: 1,
		Theme:   "system",
		Locale:  "en",
	}

	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return currentSettings, nil
		},
		UpdateFunc: func(ctx context.Context, settings *models.AppSettings) error {
			utils.Equal(t, settings.ID, uint(1))
			utils.Equal(t, settings.Theme, "dark")
			utils.Equal(t, settings.Locale, "fr")
			return nil
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	updatedSettings, err := service.Update(ctx, "dark", "fr")
	utils.NilError(t, err)
	utils.Equal(t, updatedSettings.Theme, "dark")
	utils.Equal(t, updatedSettings.Locale, "fr")
	utils.Equal(t, updatedSettings.ID, uint(1))
}

func TestAppSettingsService_Update_EmptyTheme(t *testing.T) {
	mockRepo := &mocks.AppSettingsRepositoryMock{}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Update(ctx, "", "en")
	utils.Equal(t, err.Error(), "theme is required")
}

func TestAppSettingsService_Update_EmptyLocale(t *testing.T) {
	mockRepo := &mocks.AppSettingsRepositoryMock{}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Update(ctx, "dark", "")
	utils.Equal(t, err.Error(), "locale is required")
}

func TestAppSettingsService_Update_InvalidTheme(t *testing.T) {
	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return &models.AppSettings{
				ID:      1,
				Version: 1,
				Theme:   "system",
				Locale:  "en",
			}, nil
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Update(ctx, "invalid", "en")
	utils.Equal(t, err.Error(), "theme must be 'light', 'dark', or 'system'")
}

func TestAppSettingsService_Update_GetError(t *testing.T) {
	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return nil, errors.New("get error")
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Update(ctx, "dark", "en")
	utils.Equal(t, err.Error(), "get error")
}

func TestAppSettingsService_Update_UpdateError(t *testing.T) {
	currentSettings := &models.AppSettings{
		ID:      1,
		Version: 1,
		Theme:   "system",
		Locale:  "en",
	}

	mockRepo := &mocks.AppSettingsRepositoryMock{
		GetFunc: func(ctx context.Context) (*models.AppSettings, error) {
			return currentSettings, nil
		},
		UpdateFunc: func(ctx context.Context, settings *models.AppSettings) error {
			return errors.New("update error")
		},
	}
	service := services.NewAppSettingsService(mockRepo)
	ctx := context.Background()

	_, err := service.Update(ctx, "dark", "fr")
	utils.Equal(t, err.Error(), "update error")
}
