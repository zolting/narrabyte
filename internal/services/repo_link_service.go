package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
	"narrabyte/internal/utils"
)

const llmInstructionsBaseName = "llm_instructions"

type RepoLinkService interface {
	Register(projectName, documentationRepo, codebaseRepo string) (*models.RepoLink, error)
	Get(id uint) (*models.RepoLink, error)
	List(limit, offset int) ([]models.RepoLink, error)
	LinkRepositories(projectName, docRepo, codebaseRepo string, initFumaDocs bool, llmInstructionsPath string) error
	Startup(ctx context.Context)
}

type repoLinkService struct {
	repoLinks       repositories.RepoLinkRepository
	fumadocsService FumadocsService
	gitService      GitService
	context         context.Context
}

func (s *repoLinkService) Startup(ctx context.Context) {
	s.context = ctx
}

func NewRepoLinkService(repoLinks repositories.RepoLinkRepository, fumaDocsService FumadocsService, gitService GitService) RepoLinkService {
	return &repoLinkService{repoLinks: repoLinks, fumadocsService: fumaDocsService, gitService: gitService}
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

	narrabyteDir := filepath.Join(documentationRepo, ".narrabyte")
	if st, err := os.Stat(narrabyteDir); err == nil {
		if !st.IsDir() {
			return nil, fmt.Errorf(".narrabyte exists but is not a directory")
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(narrabyteDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create .narrabyte directory: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unable to access .narrabyte directory: %w", err)
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
func (s *repoLinkService) LinkRepositories(projectName string, docRepo string, codebaseRepo string, initFumaDocs bool, llmInstructionsPath string) error {
	if s == nil {
		return fmt.Errorf("repo link service not available")
	}

	if initFumaDocs {
		_, err := s.fumadocsService.CreateFumadocsProject(docRepo)
		if err != nil {
			return fmt.Errorf("failed to create fumadocs project: %w", err)
		}
		_, err := s.gitService.Init(docRepo)
		if err != nil {
			return fmt.Errorf("failed to init git in doc repo: %w", err)
		}
	}

	_, err := s.Register(projectName, docRepo, codebaseRepo)
	if err != nil {
		return err
	}

	if llmInstructionsPath != "" {
		if err := s.storeLLMInstructions(docRepo, llmInstructionsPath); err != nil {
			return fmt.Errorf("failed to store llm instructions: %w", err)
		}
	}

	return nil
}

func (s *repoLinkService) storeLLMInstructions(docRepo, srcPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("cannot stat llm instructions file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("llm instructions path is a directory, expected file")
	}

	destDir := filepath.Join(docRepo, ".narrabyte")
	if _, err := os.Stat(destDir); err != nil {
		return fmt.Errorf("destination .narrabyte directory missing: %w", err)
	}

	ext := filepath.Ext(srcPath)
	destFile := filepath.Join(destDir, llmInstructionsBaseName+ext)

	if err := copyFile(srcPath, destFile); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	if err = out.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err = os.Chmod(dst, 0o644); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}
