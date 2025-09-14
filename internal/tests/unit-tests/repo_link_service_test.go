// go
package unit_tests

import (
	"context"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoLinkService_Register_Success(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{
		CreateFunc: func(ctx context.Context, link *models.RepoLink) error {
			link.ID = 99
			return nil
		},
	}
	fumaTest := services.FumadocsService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest)

	ctx := context.Background()
	service.Startup(ctx)

	//Create temporary directories for testing
	docDir := t.TempDir()
	codeDir := t.TempDir()

	// Create a .git directory in each temp directory to simulate a git repo
	err := os.Mkdir(docDir+string(os.PathSeparator)+".git", 0755)
	assert.NoError(t, err)
	err = os.Mkdir(codeDir+string(os.PathSeparator)+".git", 0755)
	assert.NoError(t, err)

	link, err := service.Register("name", docDir, codeDir)
	assert.NoError(t, err)
	assert.Equal(t, uint(99), link.ID)
	assert.Equal(t, "name", link.ProjectName)
	assert.Equal(t, docDir, link.DocumentationRepo)
	assert.Equal(t, codeDir, link.CodebaseRepo)
}

func TestRepoLinkService_Register_MissingDocumentationRepo(t *testing.T) {
	fumaTest := services.FumadocsService{}
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	service := services.NewRepoLinkService(mockRepo, fumaTest)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("name", "", "code")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "documentation repo is required", err.Error())
}

func TestRepoLinkService_Register_MissingCodebaseRepo(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("name", "docs", "")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "codebase repo is required", err.Error())
}

func TestRepoLinkService_Register_MissingProjectName(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("", "docs", "code")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "project name is required", err.Error())
}
