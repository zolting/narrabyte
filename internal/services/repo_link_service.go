package services

import (
	"context"
	"errors"
	"fmt"
	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
	"narrabyte/internal/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type RepoLinkService interface {
	Register(projectName, documentationRepo, codebaseRepo string) (*models.RepoLink, error)
	Get(id uint) (*models.RepoLink, error)
	List(limit, offset int) ([]models.RepoLink, error)
	Startup(ctx context.Context)
}

type repoLinkService struct {
	repoLinks       repositories.RepoLinkRepository
	fumadocsService FumadocsService
	context         context.Context
}

func (s *repoLinkService) Startup(ctx context.Context) {
	s.context = ctx
}

func NewRepoLinkService(repoLinks repositories.RepoLinkRepository, fumaDocsService FumadocsService) RepoLinkService {
	return &repoLinkService{repoLinks: repoLinks, fumadocsService: fumaDocsService}
}

func (s *repoLinkService) Register(projectName, documentationRepo, codebaseRepo string) (*models.RepoLink, error) {
	if projectName == "" {
		return nil, errors.New("project name is required")
	}

	if documentationRepo == "" {
		return nil, errors.New("documentation repo is required")
	}

	if codebaseRepo == "" {
		return nil, errors.New("codebase repo is required")
	}

	if !utils.HasGitRepo(documentationRepo) {
		return nil, errors.New("missing_git_repo: documentation")
	}

	if !utils.HasGitRepo(codebaseRepo) {
		return nil, errors.New("missing_git_repo: codebase")
	}

	if !utils.DirectoryExists(documentationRepo) {
		return nil, errors.New("documentation repo path does not exist")
	}

	if !utils.DirectoryExists(codebaseRepo) {
		return nil, errors.New("codebase repo path does not exist")
	}

	link := &models.RepoLink{
		ProjectName:       projectName,
		DocumentationRepo: documentationRepo,
		CodebaseRepo:      codebaseRepo,
	}
	if err := s.repoLinks.Create(context.Background(), link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *repoLinkService) Get(id uint) (*models.RepoLink, error) {
	return s.repoLinks.FindByID(context.Background(), id)
}

func (s *repoLinkService) List(limit, offset int) ([]models.RepoLink, error) {
	return s.repoLinks.List(context.Background(), limit, offset)
}

// LinkRepositories links the given repositories
func (s *repoLinkService) LinkRepositories(projectName, docRepo, codebaseRepo string) error {
	if s == nil {
		return fmt.Errorf("repo link service not available")
	}

	_, err := s.Register(projectName, docRepo, codebaseRepo)
	if err != nil {
		runtime.LogError(s.context, fmt.Sprintf("failed to link repositories: %v", err))
		return err
	}

	x, err := s.fumadocsService.CreateFumadocsProject(docRepo)
	if err != nil {
		runtime.LogError(context.Background(), fmt.Sprintf("failed to create fumadocs project: %v", err))
		return fmt.Errorf("failed to create fumadocs project: %w", err)
	}
	runtime.LogInfo(s.context, x)

	runtime.LogInfo(s.context, fmt.Sprintf("Successfully linked project: %s, doc: %s with codebase: %s", projectName, docRepo, codebaseRepo))
	return nil
}
