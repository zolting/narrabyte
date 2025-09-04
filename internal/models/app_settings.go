package models

import "time"

type AppSettings struct {
	ID        uint   `gorm:"primaryKey"` // single-row table (ID=1)
	Version   int    `gorm:"not null;default:1"`
	Theme     string `gorm:"not null;default:system"` // "light" | "dark" | "system"
	Locale    string `gorm:"not null"`
	UpdatedAt time.Time
}
