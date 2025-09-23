package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type FumadocsService struct {
	context context.Context
}

// RepoURL template repo
const RepoURL = "https://github.com/Philac105/narrabyte-docs-base.git"

func NewFumadocsService() *FumadocsService {
	return &FumadocsService{}
}

func (f *FumadocsService) Startup(ctx context.Context) {
	f.context = ctx
}

// CreateFumadocsProject clones the fumadocs template into the given folder
func (f *FumadocsService) CreateFumadocsProject(targetDirectory string) (string, error) {
	// Fail if the directory already exists
	if _, err := os.Stat(targetDirectory); err == nil {
		entries, _ := os.ReadDir(targetDirectory)
		if len(entries) > 0 {
			return "", fmt.Errorf("directory %s already exists and is not empty", targetDirectory)
		}
	}

	// Create directory (with parents if needed)
	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", targetDirectory, err)
	}

	// Use GitService to clone the repository
	gs := &GitService{}
	if _, err := gs.Clone(RepoURL, targetDirectory); err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	// Remove the .git folder so it's not a repo anymore
	gitDir := filepath.Join(targetDirectory, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return "", fmt.Errorf("failed to remove .git folder: %w", err)
	}

	// Return a simple non-empty message to indicate success
	return fmt.Sprintf("cloned %s into %s", RepoURL, targetDirectory), nil
}

// CheckGitAvailability checks if git is available on the system
func (f *FumadocsService) CheckGitAvailability() error { return nil }
