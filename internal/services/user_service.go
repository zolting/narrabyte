package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
)

type UserService interface {
	Register(name string) (*models.User, error)
	Get(id uint) (*models.User, error)
	List(limit, offset int) ([]models.User, error)
	Startup(ctx context.Context)
}

type userService struct {
	users   repositories.UserRepository
	context context.Context
}

func (s *userService) Startup(ctx context.Context) {
	s.context = ctx
}

func NewUserService(users repositories.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) Register(name string) (*models.User, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	u := &models.User{
		Name: name,
	}
	if err := s.users.Create(context.Background(), u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userService) Get(id uint) (*models.User, error) {
	return s.users.FindByID(context.Background(), id)
}

func (s *userService) List(limit, offset int) ([]models.User, error) {
	return s.users.List(context.Background(), limit, offset)
}

// Greet returns a greeting for the given name
func (s *userService) Greet(name string) string {
	if name != "" {
		if _, err := s.Register(name); err != nil {
			runtime.LogError(s.context, fmt.Sprintf("failed to create user: %v", err))
		}
	}

	return fmt.Sprintf("Hello bruvddd %s, It's show time!", name)
}
