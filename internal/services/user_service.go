package service

import (
	"context"
	"narrabyte/internal/models"
	"narrabyte/internal/repository"
)

type UserService interface {
	Register(ctx context.Context, name string) (*models.User, error)
	Get(ctx context.Context, id uint) (*models.User, error)
	List(ctx context.Context, limit, offset int) ([]models.User, error)
}

type userService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) Register(ctx context.Context, name string) (*models.User, error) {
	u := &models.User{
		Name: name,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userService) Get(ctx context.Context, id uint) (*models.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *userService) List(ctx context.Context, limit, offset int) ([]models.User, error) {
	return s.users.List(ctx, limit, offset)
}
