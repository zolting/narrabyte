package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	ctx := context.Background()

	// Check if git is installed
	if err := f.CheckGitAvailability(ctx); err != nil {
		return "git is not installed", err
	}

	projectPath := filepath.Join(targetFolder, ProjectName)

	// Fail if the directory already exists
	if _, err := os.Stat(projectPath); err == nil {
		return "", fmt.Errorf("directory %s already exists", projectPath)
	}

	// Clone the repo into projectPath
	cmd := exec.CommandContext(ctx, "git", "clone", RepoURL, projectPath)
	cmd.Dir = targetFolder
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to clone repo: %w", err)
	}

	// Remove the .git folder so it's not a repo anymore
	gitDir := filepath.Join(projectPath, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return string(output), fmt.Errorf("failed to remove .git folder: %w", err)
	}

	return string(output), nil
}

// CheckGitAvailability checks if git is available on the system
func (f *FumadocsService) CheckGitAvailability(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s is unavailable: %w", "git", err)
	}
	return nil
}
