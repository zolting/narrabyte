package unit_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"

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
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	//Create temporary directories for testing
	docDir := t.TempDir()
	codeDir := t.TempDir()

	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	link, err := service.Register("name", docDir, codeDir, "main")
	assert.NoError(t, err)
	assert.Equal(t, uint(99), link.ID)
	assert.Equal(t, "main", link.DocumentationBaseBranch)

	// Assert .narrabyte directory created
	ni := filepath.Join(docDir, ".narrabyte")
	st, statErr := os.Stat(ni)
	assert.NoError(t, statErr)
	assert.True(t, st.IsDir())
}

func TestRepoLinkService_Register_MissingGitRepo(t *testing.T) {
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("name", "docs", "code", "main")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "missing_git_repo: documentation", err.Error())
}

func TestRepoLinkService_Register_MissingDocumentationRepo(t *testing.T) {
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("name", "", "code", "main")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "documentation repo is required", err.Error())
}

func TestRepoLinkService_Register_MissingCodebaseRepo(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("name", "docs", "", "main")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "codebase repo is required", err.Error())
}

func TestRepoLinkService_Register_MissingProjectName(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	link, err := service.Register("", "docs", "code", "main")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "project name is required", err.Error())
}

func TestRepoLinkService_Register_MissingBaseBranchForSeparateRepos(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	docDir := t.TempDir()
	codeDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	link, err := service.Register("name", docDir, codeDir, "")
	assert.Nil(t, link)
	assert.Error(t, err)
	assert.Equal(t, "documentation base branch is required", err.Error())
}

func TestRepoLinkService_Register_SharedRepoAllowsEmptyBaseBranch(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{
		CreateFunc: func(ctx context.Context, link *models.RepoLink) error {
			link.ID = 42
			return nil
		},
	}
	service := services.NewRepoLinkService(mockRepo, services.FumadocsService{}, services.GitService{})
	service.Startup(context.Background())

	sharedDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(sharedDir, ".git"), 0o755))

	link, err := service.Register("name", sharedDir, sharedDir, "")
	assert.NoError(t, err)
	assert.NotNil(t, link)
	assert.Equal(t, uint(42), link.ID)
	assert.Equal(t, "", link.DocumentationBaseBranch)
}

func TestRepoLinkService_LinkRepositories_WithLLMInstructions(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	docDir := t.TempDir()
	codeDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	// Create instructions file
	instFile := filepath.Join(t.TempDir(), "custom.md")
	content := []byte("Project specific LLM instructions.")
	assert.NoError(t, os.WriteFile(instFile, content, 0o644))

	err := service.LinkRepositories("proj", docDir, codeDir, false, instFile, "main")
	assert.NoError(t, err)

	// Verify copied
	copied := filepath.Join(docDir, ".narrabyte", "llm_instructions.md")
	data, rErr := os.ReadFile(copied)
	assert.NoError(t, rErr)
	assert.Equal(t, string(content), string(data))
}

func TestRepoLinkService_LinkRepositories_NoLLMInstructions(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{}
	fumaTest := services.FumadocsService{}
	gitSvc := services.GitService{}
	service := services.NewRepoLinkService(mockRepo, fumaTest, gitSvc)

	ctx := context.Background()
	service.Startup(ctx)

	docDir := t.TempDir()
	codeDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	err := service.LinkRepositories("proj", docDir, codeDir, false, "", "main")
	assert.NoError(t, err)

	// .narrabyte still created
	_, statErr := os.Stat(filepath.Join(docDir, ".narrabyte"))
	assert.NoError(t, statErr)
}

func TestRepoLinkService_LinkRepositories_LLMInstructionsIsDirectory(t *testing.T) {
	mockRepo := &mocks.RepoLinkRepositoryMock{
		CreateFunc: func(ctx context.Context, link *models.RepoLink) error { return nil },
	}
	service := services.NewRepoLinkService(mockRepo, services.FumadocsService{}, services.GitService{})
	service.Startup(context.Background())

	docDir := t.TempDir()
	codeDir := t.TempDir()
	instDir := t.TempDir()

	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	err := service.LinkRepositories("proj", docDir, codeDir, false, instDir, "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store llm instructions")
}

func TestRepoLinkService_UpdateProjectPaths_RequiresBaseBranch(t *testing.T) {
	docDir := t.TempDir()
	codeDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	repo := &mocks.RepoLinkRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint) (*models.RepoLink, error) {
			return &models.RepoLink{
				ID:                      id,
				ProjectName:             "proj",
				DocumentationRepo:       docDir,
				CodebaseRepo:            codeDir,
				DocumentationBaseBranch: "main",
			}, nil
		},
	}
	service := services.NewRepoLinkService(repo, services.FumadocsService{}, services.GitService{})
	service.Startup(context.Background())

	err := service.UpdateProjectPaths(1, "", "", "")
	assert.Error(t, err)
	assert.Equal(t, "documentation base branch is required", err.Error())
}

func TestRepoLinkService_UpdateProjectPaths_UpdatesBaseBranch(t *testing.T) {
	docDir := t.TempDir()
	codeDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(docDir, ".git"), 0o755))
	assert.NoError(t, os.Mkdir(filepath.Join(codeDir, ".git"), 0o755))

	var updated *models.RepoLink
	repo := &mocks.RepoLinkRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint) (*models.RepoLink, error) {
			return &models.RepoLink{
				ID:                      id,
				ProjectName:             "proj",
				DocumentationRepo:       docDir,
				CodebaseRepo:            codeDir,
				DocumentationBaseBranch: "main",
			}, nil
		},
		UpdateFunc: func(ctx context.Context, link *models.RepoLink) error {
			updated = link
			return nil
		},
	}
	service := services.NewRepoLinkService(repo, services.FumadocsService{}, services.GitService{})
	service.Startup(context.Background())

	err := service.UpdateProjectPaths(2, "", "", " docs ")
	assert.NoError(t, err)
	if assert.NotNil(t, updated) {
		assert.Equal(t, "docs", updated.DocumentationBaseBranch)
	}
}
