package services

import (
	context "context"
	"fmt"

	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
)

type TemplateService interface {
	GetTemplate(id uint) (*models.Template, error)
	ListTemplates() ([]*models.Template, error)
	CreateTemplate(t *models.Template) (*models.Template, error)
	UpdateTemplate(t *models.Template) (*models.Template, error)
	DeleteTemplate(id uint) error
	Startup(ctx context.Context)
}

type templateService struct {
	repo repositories.TemplateRepository
	ctx  context.Context
}

func (s *templateService) Startup(ctx context.Context) {
	s.ctx = ctx
}

func NewTemplateService(repo repositories.TemplateRepository) TemplateService {
	return &templateService{repo: repo}
}

func (s *templateService) GetTemplate(id uint) (*models.Template, error) {
	tmpl, err := s.repo.Get(s.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get template %d: %w", id, err)
	}
	return tmpl, nil
}

func (s *templateService) ListTemplates() ([]*models.Template, error) {
	list, err := s.repo.GetAll(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("service: list templates: %w", err)
	}
	return list, nil
}

func (s *templateService) CreateTemplate(t *models.Template) (*models.Template, error) {
	if err := s.repo.Create(s.ctx, t); err != nil {
		return nil, fmt.Errorf("service: create template: %w", err)
	}
	return t, nil
}

func (s *templateService) UpdateTemplate(t *models.Template) (*models.Template, error) {
	if err := s.repo.Update(s.ctx, t); err != nil {
		return nil, fmt.Errorf("service: update template %d: %w", t.ID, err)
	}
	return t, nil
}

func (s *templateService) DeleteTemplate(id uint) error {
	if err := s.repo.Delete(s.ctx, id); err != nil {
		return fmt.Errorf("service: delete template %d: %w", id, err)
	}
	return nil
}
