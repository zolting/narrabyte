package repositories

import (
	"context"
	"narrabyte/internal/models"

	"gorm.io/gorm"
)

type RepoLinkRepository interface {
	Create(ctx context.Context, link *models.RepoLink) error
	FindByID(ctx context.Context, id uint) (*models.RepoLink, error)
	List(ctx context.Context, limit, offset int) ([]models.RepoLink, error)
}

type repoLinkRepository struct {
	db *gorm.DB
}

func NewRepoLinkRepository(db *gorm.DB) RepoLinkRepository {
	return &repoLinkRepository{db: db}
}

func (r *repoLinkRepository) Create(ctx context.Context, link *models.RepoLink) error {
	return r.db.WithContext(ctx).Create(link).Error
}

func (r *repoLinkRepository) FindByID(ctx context.Context, id uint) (*models.RepoLink, error) {
	var link models.RepoLink
	err := r.db.WithContext(ctx).First(&link, id).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *repoLinkRepository) List(ctx context.Context, limit, offset int) ([]models.RepoLink, error) {
	var links []models.RepoLink
	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&links).Error
	return links, err
}
