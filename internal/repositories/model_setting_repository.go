package repositories

import (
	"fmt"
	"narrabyte/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ModelSettingRepository interface {
	List() ([]models.ModelSetting, error)
	GetByKey(modelKey string) (*models.ModelSetting, error)
	Upsert(modelKey, provider string, enabled bool) (*models.ModelSetting, error)
	SetProviderEnabled(provider string, enabled bool) error
}

type modelSettingRepository struct {
	db *gorm.DB
}

func NewModelSettingRepository(db *gorm.DB) ModelSettingRepository {
	return &modelSettingRepository{db: db}
}

func (r *modelSettingRepository) List() ([]models.ModelSetting, error) {
	var settings []models.ModelSetting
	if err := r.db.Order("provider, model_key").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *modelSettingRepository) GetByKey(modelKey string) (*models.ModelSetting, error) {
	if modelKey == "" {
		return nil, fmt.Errorf("model key is required")
	}
	var setting models.ModelSetting
	if err := r.db.Where("model_key = ?", modelKey).Take(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func (r *modelSettingRepository) Upsert(modelKey, provider string, enabled bool) (*models.ModelSetting, error) {
	if modelKey == "" {
		return nil, fmt.Errorf("model key is required")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	record := models.ModelSetting{
		ModelKey: modelKey,
		Provider: provider,
		Enabled:  enabled,
	}
	if err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "model_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"enabled":    enabled,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *modelSettingRepository) SetProviderEnabled(provider string, enabled bool) error {
	if provider == "" {
		return fmt.Errorf("provider is required")
	}
	return r.db.Model(&models.ModelSetting{}).
		Where("provider = ?", provider).
		Update("enabled", enabled).Error
}
