package models

import "time"

type GenerationSession struct {
	ID               uint   `gorm:"primaryKey"`
	ProjectID        uint   `gorm:"index:idx_session_project_docs,unique"`
	SourceBranch     string `gorm:"size:255;not null"`
	TargetBranch     string `gorm:"size:255;not null"`
	Provider         string `gorm:"size:50;not null"`
	ModelKey         string `gorm:"size:255"`
	DocsBranch       string `gorm:"size:255;index:idx_session_project_docs,unique"`
	MessagesJSON     string `gorm:"type:text"`
	ChatMessagesJSON string `gorm:"type:text"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
