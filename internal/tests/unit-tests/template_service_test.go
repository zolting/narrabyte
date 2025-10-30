package unit_tests

import (
	"context"
	"testing"

	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"

	"github.com/stretchr/testify/assert"
)

func TestTemplateService_CreateTemplate_Success(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		CreateFunc: func(ctx context.Context, tmpl *models.Template) error {
			tmpl.ID = 42
			return nil
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	tmpl := &models.Template{Name: "Test", Content: "Content"}
	result, err := service.CreateTemplate(tmpl)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), result.ID)
}

func TestTemplateService_CreateTemplate_Error(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		CreateFunc: func(ctx context.Context, tmpl *models.Template) error {
			return assert.AnError
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	tmpl := &models.Template{Name: "Test", Content: "Content"}
	result, err := service.CreateTemplate(tmpl)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTemplateService_GetTemplate_Success(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		GetFunc: func(ctx context.Context, id uint) (*models.Template, error) {
			return &models.Template{ID: id, Name: "Test", Content: "Content"}, nil
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	result, err := service.GetTemplate(1)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), result.ID)
}

func TestTemplateService_GetTemplate_Error(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		GetFunc: func(ctx context.Context, id uint) (*models.Template, error) {
			return nil, assert.AnError
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	result, err := service.GetTemplate(1)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTemplateService_ListTemplates_Success(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		GetAllFunc: func(ctx context.Context) ([]*models.Template, error) {
			return []*models.Template{
				{ID: 1, Name: "A", Content: "A"},
				{ID: 2, Name: "B", Content: "B"},
			}, nil
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	result, err := service.ListTemplates()
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTemplateService_ListTemplates_Error(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		GetAllFunc: func(ctx context.Context) ([]*models.Template, error) {
			return nil, assert.AnError
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	result, err := service.ListTemplates()
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTemplateService_UpdateTemplate_Success(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		UpdateFunc: func(ctx context.Context, tmpl *models.Template) error {
			return nil
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	tmpl := &models.Template{ID: 1, Name: "Updated", Content: "Updated"}
	result, err := service.UpdateTemplate(tmpl)
	assert.NoError(t, err)
	assert.Equal(t, tmpl, result)
}

func TestTemplateService_UpdateTemplate_Error(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		UpdateFunc: func(ctx context.Context, tmpl *models.Template) error {
			return assert.AnError
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	tmpl := &models.Template{ID: 1, Name: "Updated", Content: "Updated"}
	result, err := service.UpdateTemplate(tmpl)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTemplateService_DeleteTemplate_Success(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		DeleteFunc: func(ctx context.Context, id uint) error {
			return nil
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	err := service.DeleteTemplate(1)
	assert.NoError(t, err)
}

func TestTemplateService_DeleteTemplate_Error(t *testing.T) {
	mockRepo := &mocks.TemplateRepositoryMock{
		DeleteFunc: func(ctx context.Context, id uint) error {
			return assert.AnError
		},
	}
	service := services.NewTemplateService(mockRepo)
	service.Startup(context.Background())

	err := service.DeleteTemplate(1)
	assert.Error(t, err)
}
