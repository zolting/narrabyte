package models

import "time"

// ModelSetting persists per-model enablement toggles.
type ModelSetting struct {
	ID        uint      `gorm:"primaryKey"`
	Provider  string    `gorm:"size:50;not null;index:idx_model_provider"`
	ModelKey  string    `gorm:"size:255;not null;uniqueIndex"`
	Enabled   bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
