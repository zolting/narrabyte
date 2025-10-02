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
	Update(ctx context.Context, link *models.RepoLink) error
	Delete(ctx context.Context, id uint) error
	UpdateOrder(ctx context.Context, updates []models.RepoLinkOrderUpdate) error
	IncrementAllIndexes(ctx context.Context) error
	GetMaxIndex(ctx context.Context) (int, error)
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
	err := r.db.WithContext(ctx).
		Order("`index` ASC").
		Limit(limit).
		Offset(offset).
		Find(&links).
		Error
	return links, err
}

func (r *repoLinkRepository) Update(ctx context.Context, link *models.RepoLink) error {
	return r.db.WithContext(ctx).Save(link).Error
}

func (r *repoLinkRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RepoLink{}, id).Error
}

func (r *repoLinkRepository) UpdateOrder(ctx context.Context, updates []models.RepoLinkOrderUpdate) error {
	tx := r.db.WithContext(ctx).Begin()

	for _, update := range updates {
		if err := tx.Model(&models.RepoLink{}).
			Where("id = ?", update.ID).
			Update("index", update.Index).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (r *repoLinkRepository) IncrementAllIndexes(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&models.RepoLink{}).
		Where("1 = 1").
		Update("`index`", gorm.Expr("`index` + ?", 1)).
		Error
}

func (r *repoLinkRepository) GetMaxIndex(ctx context.Context) (int, error) {
	var maxIndex int
	err := r.db.WithContext(ctx).
		Model(&models.RepoLink{}).
		Select("COALESCE(MAX(index), -1)").
		Scan(&maxIndex).
		Error
	return maxIndex, err
}
