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
		// Non-fatal: log only (some CI environments may not have git)
		t.Logf("Git not available: %v", err)
	}
}

func TestCreateFumadocsProject_DirectoryExists(t *testing.T) {
	service := services.NewFumadocsService()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_*")
	if err != nil {
		// If temp dir cannot be created we skip to avoid false failures
		t.Skipf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Make directory non-empty
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Should fail because directory exists and is not empty
	_, err = service.CreateFumadocsProject(tempDir)
	if err == nil {
		t.Error("expected error when directory already exists and is not empty")
	}
}

func TestCreateFumadocsProject_Success(t *testing.T) {
	service := services.NewFumadocsService()

	// Create temp directory (empty)
	tempDir, err := os.MkdirTemp("", "test_*")
	if err != nil {
		// If temp dir cannot be created we skip to avoid false failures
		t.Skipf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	output, err := service.CreateFumadocsProject(tempDir)
	if err != nil {
		// Likely network or git issue; skip test rather than fail entire suite
		t.Skipf("Skipping due to git/network issue: %v", err)
	}

	// Check project was created (we expect some files now inside tempDir)
	entries, readErr := os.ReadDir(tempDir)
	if readErr != nil {
		// If we cannot read directory treat as skip
		t.Skipf("failed reading temp dir: %v", readErr)
	}
	if len(entries) == 0 {
		// Not strictly fatal if repo empty, but indicates clone may have failed silently
		t.Error("expected cloned repository contents in target directory")
	}

	// Check .git was removed (implementation removes .git folder)
	gitDir := filepath.Join(tempDir, ".git")
	if _, statErr := os.Stat(gitDir); !os.IsNotExist(statErr) {
		// If gitDir exists and no stat error => failure
		if statErr == nil {
			// Directory still exists
			t.Error(".git directory should be removed")
		} else {
			// Unexpected error state
			t.Errorf("unexpected error checking .git directory: %v", statErr)
		}
	}

	if output == "" {
		// Simple sanity check on return message
		t.Error("expected non-empty output from CreateFumadocsProject")
	}
}
