package models

import (
	"time"
)

type User struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Name      string `gorm:"size:120"`
	AvatarURL string `gorm:"size:512"`
}
