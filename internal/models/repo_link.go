package models

type RepoLink struct {
	ID                      uint `gorm:"primaryKey"`
	DocumentationRepo       string
	CodebaseRepo            string
	ProjectName             string
	DocumentationBaseBranch string
	Index                   int `json:"index"`
}

type RepoLinkOrderUpdate struct {
	ID    uint `json:"ID"`
	Index int  `json:"Index"`
}
