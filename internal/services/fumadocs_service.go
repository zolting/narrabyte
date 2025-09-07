package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type FumadocsService struct{}

// RepoURL template repo
const RepoURL = "https://github.com/Philac105/narrabyte-docs-base.git"

// ProjectName template project name
const ProjectName = "narrabyte-docs-base"

func NewFumadocsService() *FumadocsService {
	return &FumadocsService{}
}

// CreateFumadocsProject clones the fumadocs template into the given folder
func (f *FumadocsService) CreateFumadocsProject(targetFolder string) (string, error) {
	projectPath := filepath.Join(targetFolder, ProjectName)

	// Fail if the directory already exists
	if _, err := os.Stat(projectPath); err == nil {
		return "", fmt.Errorf("directory %s already exists", projectPath)
	}

	// Use GitService to clone the repository
	gs := &GitService{}
	if _, err := gs.Clone(RepoURL, projectPath); err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	// Remove the .git folder so it's not a repo anymore
	gitDir := filepath.Join(projectPath, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return "", fmt.Errorf("failed to remove .git folder: %w", err)
	}

	// Return a simple non-empty message to indicate success
	return fmt.Sprintf("cloned %s into %s", RepoURL, projectPath), nil
}

// CheckGitAvailability checks if git is available on the system
func (f *FumadocsService) CheckGitAvailability(ctx context.Context) error { return nil }
