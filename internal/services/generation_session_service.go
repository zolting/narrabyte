package services

import (
	"context"
	"fmt"
	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
	"strings"
)

type GenerationSessionService interface {
	Startup(ctx context.Context)
	List(projectID uint) ([]models.GenerationSession, error)
	Get(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error)
	Upsert(projectID uint, sourceBranch, targetBranch, provider, messagesJSON string) (*models.GenerationSession, error)
	Delete(projectID uint, sourceBranch, targetBranch string) error
	DeleteAll(projectID uint) error
}

type generationSessionService struct {
	repo repositories.GenerationSessionRepository
	ctx  context.Context
}

func NewGenerationSessionService(repo repositories.GenerationSessionRepository) GenerationSessionService {
	return &generationSessionService{repo: repo}
}

func (s *generationSessionService) Startup(ctx context.Context) {
	s.ctx = ctx
}

func (s *generationSessionService) List(projectID uint) ([]models.GenerationSession, error) {
	return s.repo.ListByProject(projectID)
}

func (s *generationSessionService) Get(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error) {
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}

	return s.repo.GetByProjectAndBranches(projectID, sourceBranch, targetBranch)
}

func (s *generationSessionService) Upsert(projectID uint, sourceBranch, targetBranch, provider, messagesJSON string) (*models.GenerationSession, error) {
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	provider = strings.TrimSpace(provider)
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	return s.repo.Upsert(projectID, sourceBranch, targetBranch, provider, messagesJSON)
}

func (s *generationSessionService) Delete(projectID uint, sourceBranch, targetBranch string) error {
	sourceBranch = strings.TrimSpace(sourceBranch)
	targetBranch = strings.TrimSpace(targetBranch)
	if sourceBranch == "" || targetBranch == "" {
		return fmt.Errorf("source and target branches are required")
	}

	return s.repo.DeleteByProjectAndBranches(projectID, sourceBranch, targetBranch)
}

func (s *generationSessionService) DeleteAll(projectID uint) error {
	return s.repo.DeleteByProject(projectID)
}
