package models

type RepoLink struct {
	ID                uint `gorm:"primaryKey"`
	DocumentationRepo string
	CodebaseRepo      string
	ProjectName       string
}
