package repository

import (
	"context"

	"gorm.io/gorm"

	"narrabyte/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	List(ctx context.Context, limit, offset int) ([]models.User, error)
	Update(ctx context.Context, u *models.User) error
	Delete(ctx context.Context, id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	var users []models.User
	q := r.db.WithContext(ctx).Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) Update(ctx context.Context, u *models.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}
