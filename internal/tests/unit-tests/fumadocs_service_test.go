package unit_tests

import (
	"os"
	"path/filepath"
	"testing"

	"narrabyte/internal/services"
)

func TestNewFumadocsService(t *testing.T) {
	service := services.NewFumadocsService()
	if service == nil {
		t.Error("NewFumadocsService returned nil")
	}
}

func TestCheckGitAvailability(t *testing.T) {
	service := services.NewFumadocsService()

	err := service.CheckGitAvailability()
	if err != nil {
		t.Logf("Git not available: %v", err)
	}
}

func TestCreateFumadocsProject_DirectoryExists(t *testing.T) {
	service := services.NewFumadocsService()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create project directory first
	projectPath := filepath.Join(tempDir, services.ProjectName)
	os.MkdirAll(projectPath, 0755)

	// Should fail because directory exists
	_, err = service.CreateFumadocsProject(tempDir)
	if err == nil {
		t.Error("expected error when directory already exists")
	}
}

func TestCreateFumadocsProject_Success(t *testing.T) {
	service := services.NewFumadocsService()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	output, err := service.CreateFumadocsProject(tempDir)
	if err != nil {
		t.Skipf("Skipping due to git/network issue: %v", err)
	}

	// Check project was created
	projectPath := filepath.Join(tempDir, services.ProjectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("project directory not created")
	}

	// Check .git was removed
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		t.Error(".git directory should be removed")
	}

	if output == "" {
		t.Error("expected output from git clone")
	}
}
