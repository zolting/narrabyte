package unit_tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"testing"
)

func TestRepoLinkService_Register_Success(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{
		CreateFunc: func(ctx context.Context, link *models.RepoLink) error {
			link.ID = 99
			return nil
		},
	}
	service := services.NewRepoLinkService(mockRepo)
	ctx := context.Background()

	link, err := service.Register(ctx, "docs", "code")
	assert.NoError(t, err)
	assert.Equal(t, uint(99), link.ID)
	assert.Equal(t, "docs", link.DocumentationRepo)
	assert.Equal(t, "code", link.CodebaseRepo)
}

func TestRepoLinkService_Register_MissingDocumentationRepo(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	service := services.NewRepoLinkService(mockRepo)
	ctx := context.Background()

	link, err := service.Register(ctx, "", "code")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "documentation repo is required", err.Error())
}

func TestRepoLinkService_Register_MissingCodebaseRepo(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	service := services.NewRepoLinkService(mockRepo)
	ctx := context.Background()

	link, err := service.Register(ctx, "docs", "")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "codebase repo is required", err.Error())
}
