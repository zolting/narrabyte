package repositories

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"narrabyte/internal/models"
)

type TemplateRepository interface {
	Get(ctx context.Context, id uint) (*models.Template, error)
	GetAll(ctx context.Context) ([]*models.Template, error)
	Create(ctx context.Context, template *models.Template) error
	Update(ctx context.Context, template *models.Template) error
	Delete(ctx context.Context, id uint) error
}

type templateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Get(ctx context.Context, id uint) (*models.Template, error) {
	var tmpl models.Template
	if err := r.db.WithContext(ctx).First(&tmpl, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("getting template %d: %w", id, err)
	}
	return &tmpl, nil
}

func (r *templateRepository) GetAll(ctx context.Context) ([]*models.Template, error) {
	var list []*models.Template
	if err := r.db.WithContext(ctx).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	return list, nil
}

func (r *templateRepository) Create(ctx context.Context, template *models.Template) error {
	if err := r.db.WithContext(ctx).Create(template).Error; err != nil {
		return fmt.Errorf("creating template: %w", err)
	}
	return nil
}

func (r *templateRepository) Update(ctx context.Context, template *models.Template) error {
	if err := r.db.WithContext(ctx).Save(template).Error; err != nil {
		return fmt.Errorf("updating template %d: %w", template.ID, err)
	}
	return nil
}

func (r *templateRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Template{}, id).Error; err != nil {
		return fmt.Errorf("deleting template %d: %w", id, err)
	}
	return nil
}
