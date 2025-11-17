package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"narrabyte/internal/models"
)

type AppSettingsRepository interface {
	Get(ctx context.Context) (*models.AppSettings, error)
	Update(ctx context.Context, settings *models.AppSettings) error
}

type appSettingsRepository struct {
	db *gorm.DB
}

func NewAppSettingsRepository(db *gorm.DB) AppSettingsRepository {
	return &appSettingsRepository{db: db}
}

func (r *appSettingsRepository) Get(ctx context.Context) (*models.AppSettings, error) {
	var settings models.AppSettings
	if err := r.db.WithContext(ctx).First(&settings, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default settings if not found
			return &models.AppSettings{
				ID:              1,
				Version:         1,
				Theme:           "system",
				Locale:          "en",
				DefaultModelKey: "",
				UpdatedAt:       "", // empty string represents zero time
			}, nil
		}
		return nil, err
	}
	return &settings, nil
}

func (r *appSettingsRepository) Update(ctx context.Context, settings *models.AppSettings) error {
	// Ensure ID is set to 1 for single-row table
	settings.ID = 1
	return r.db.WithContext(ctx).Save(settings).Error
}
