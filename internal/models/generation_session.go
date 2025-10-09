package models

import "time"

type GenerationSession struct {
	ID           uint   `gorm:"primaryKey"`
	ProjectID    uint   `gorm:"index:idx_session_project_source_target,unique"`
	SourceBranch string `gorm:"size:255;not null;index:idx_session_project_source_target,unique"`
	TargetBranch string `gorm:"size:255;not null;index:idx_session_project_source_target,unique"`
	MessagesJSON string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
