package unit_tests

import (
	"context"
	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"narrabyte/internal/utils"
	"testing"
)

func TestUserService_Register_Success(t *testing.T) {
	mockRepo := &mocks.UserRepositoryMock{
		CreateFunc: func(ctx context.Context, u *models.User) error {
			u.ID = 42
			return nil
		},
	}
	service := services.NewUserService(mockRepo)

	user, err := service.Register("Alice")
	utils.NilError(t, err)
	utils.Equal(t, user.ID, uint(42))
	utils.Equal(t, user.Name, "Alice")
}

func TestUserService_Register_MissingName(t *testing.T) {
	mockRepo := &mocks.UserRepositoryMock{}
	service := services.NewUserService(mockRepo)

	_, err := service.Register("")
	utils.Equal(t, err.Error(), "name is required")
}
