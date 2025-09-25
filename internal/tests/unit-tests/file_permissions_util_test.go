package unit_tests

import (
	"context"
	"errors"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"narrabyte/internal/utils"
	"testing"
)

func newFileServiceWithRepoMock(findFunc func(ctx context.Context, id uint) (*models.RepoLink, error)) *services.FilePermissionsUtil {
	repoMock := &mocks.RepoLinkRepositoryMock{
		FindByIDFunc: findFunc,
	}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(repoMock, fumaTest, gitSvc)
	return &services.FilePermissionsUtil{RepoLinks: service}
}

func TestCheckReadPermissionsWithRepoMock_FilePermitted(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return &models.RepoLink{ID: 1, DocumentationRepo: "docs/", CodebaseRepo: "src/"}, nil
	})
	ok, err := fs.CheckReadPermissions(1, "src/main.go")
	utils.NilError(t, err)
	utils.Equal(t, ok, true)
}

func TestCheckReadPermissionsWithRepoMock_FileNotPermitted(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return &models.RepoLink{ID: 1, DocumentationRepo: "docs/", CodebaseRepo: "src/"}, nil
	})
	ok, err := fs.CheckReadPermissions(1, "other/file.txt")
	utils.NilError(t, err)
	utils.Equal(t, ok, false)
}

func TestCheckReadPermissionsWithRepoMock_FileDoesNotExist(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return nil, errors.New("not found")
	})
	ok, err := fs.CheckReadPermissions(1, "src/main.go")
	utils.Equal(t, err.Error(), "not found")
	utils.Equal(t, ok, false)
}

func TestCheckWritePermissionsWithRepoMock_FilePermitted(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return &models.RepoLink{ID: 1, DocumentationRepo: "docs/", CodebaseRepo: "src/"}, nil
	})
	ok, err := fs.CheckWritePermissions(1, "docs/readme.md")
	utils.NilError(t, err)
	utils.Equal(t, ok, true)
}

func TestCheckWritePermissionsWithRepoMock_FileNotPermitted(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return &models.RepoLink{ID: 1, DocumentationRepo: "docs/", CodebaseRepo: "src/"}, nil
	})
	ok, err := fs.CheckWritePermissions(1, "src/main.go")
	utils.NilError(t, err)
	utils.Equal(t, ok, false)
}

func TestCheckWritePermissionsWithRepoMock_FileDoesNotExist(t *testing.T) {
	fs := newFileServiceWithRepoMock(func(ctx context.Context, id uint) (*models.RepoLink, error) {
		return nil, errors.New("not found")
	})
	ok, err := fs.CheckWritePermissions(1, "docs/readme.md")
	utils.Equal(t, err.Error(), "not found")
	utils.Equal(t, ok, false)
}
