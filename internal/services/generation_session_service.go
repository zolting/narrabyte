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
	GetByID(id uint) (*models.GenerationSession, error)
	GetByDocsBranch(projectID uint, docsBranch string) (*models.GenerationSession, error)
	Create(session *models.GenerationSession) (*models.GenerationSession, error)
	UpdateByID(id uint, updates map[string]interface{}) error
	DeleteByID(id uint) error
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

func (s *generationSessionService) GetByID(id uint) (*models.GenerationSession, error) {
	if id == 0 {
		return nil, fmt.Errorf("session ID is required")
	}
	return s.repo.GetByID(id)
}

func (s *generationSessionService) GetByDocsBranch(projectID uint, docsBranch string) (*models.GenerationSession, error) {
	docsBranch = strings.TrimSpace(docsBranch)
	if docsBranch == "" {
		return nil, fmt.Errorf("docsBranch is required")
	}
	return s.repo.GetByDocsBranch(projectID, docsBranch)
}

func (s *generationSessionService) Create(session *models.GenerationSession) (*models.GenerationSession, error) {
	if session == nil {
		return nil, fmt.Errorf("session is required")
	}
	session.SourceBranch = strings.TrimSpace(session.SourceBranch)
	session.TargetBranch = strings.TrimSpace(session.TargetBranch)
	session.Provider = strings.TrimSpace(session.Provider)
	session.ModelKey = strings.TrimSpace(session.ModelKey)
	session.DocsBranch = strings.TrimSpace(session.DocsBranch)

	if err := s.repo.Create(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *generationSessionService) UpdateByID(id uint, updates map[string]interface{}) error {
	if id == 0 {
		return fmt.Errorf("session ID is required")
	}
	return s.repo.UpdateByID(id, updates)
}

func (s *generationSessionService) DeleteByID(id uint) error {
	if id == 0 {
		return fmt.Errorf("session ID is required")
	}
	return s.repo.DeleteByID(id)
}

func (s *generationSessionService) DeleteAll(projectID uint) error {
	return s.repo.DeleteByProject(projectID)
}
