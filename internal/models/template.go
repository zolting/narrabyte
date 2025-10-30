package models

type Template struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"size:255;not null;unique" json:"name"`
	Content string `gorm:"type:text;not null;" json:"content"`
}
