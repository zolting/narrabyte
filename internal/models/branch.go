package models

import "time"

// BranchInfo represents a git branch with its latest commit timestamp
type BranchInfo struct {
	Name           string    `json:"name"`
	LastCommitDate time.Time `json:"lastCommitDate"`
}
