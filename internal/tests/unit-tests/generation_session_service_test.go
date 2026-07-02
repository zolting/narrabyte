package unit_tests

import (
	"context"
	"errors"
	"testing"

	"narrabyte/internal/models"
	"narrabyte/internal/services"
	"narrabyte/internal/tests/mocks"
	"narrabyte/internal/utils"
)

func TestGenerationSessionService_List_Delegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.ListByProjectFunc = func(projectID uint) ([]models.GenerationSession, error) {
		utils.Equal(t, projectID, uint(42))
		return []models.GenerationSession{{ProjectID: projectID, SourceBranch: "s", TargetBranch: "t"}}, nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	sessions, err := svc.List(42)
	utils.NilError(t, err)
	utils.Equal(t, len(sessions), 1)
	utils.Equal(t, sessions[0].ProjectID, uint(42))
}

func TestGenerationSessionService_GetByID_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	_, err := svc.GetByID(0)
	utils.Equal(t, err.Error(), "session ID is required")
}

func TestGenerationSessionService_GetByID_Delegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.GetByIDFunc = func(id uint) (*models.GenerationSession, error) {
		utils.Equal(t, id, uint(7))
		return &models.GenerationSession{ID: id, ProjectID: 3}, nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	res, err := svc.GetByID(7)
	utils.NilError(t, err)
	utils.Equal(t, res.ID, uint(7))
	utils.Equal(t, res.ProjectID, uint(3))
}

func TestGenerationSessionService_GetByDocsBranch_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	_, err := svc.GetByDocsBranch(1, " ")
	utils.Equal(t, err.Error(), "docsBranch is required")
}

func TestGenerationSessionService_GetByDocsBranch_TrimsAndDelegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.GetByDocsBranchFunc = func(projectID uint, docsBranch string) (*models.GenerationSession, error) {
		utils.Equal(t, projectID, uint(7))
		utils.Equal(t, docsBranch, "docs/src")
		return &models.GenerationSession{ProjectID: projectID, DocsBranch: docsBranch}, nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	res, err := svc.GetByDocsBranch(7, "  docs/src\t")
	utils.NilError(t, err)
	utils.Equal(t, res.ProjectID, uint(7))
	utils.Equal(t, res.DocsBranch, "docs/src")
}

func TestGenerationSessionService_Create_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	_, err := svc.Create(nil)
	utils.Equal(t, err.Error(), "session is required")
}

func TestGenerationSessionService_Create_TrimsAndDelegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.CreateFunc = func(session *models.GenerationSession) error {
		utils.Equal(t, session.SourceBranch, "src")
		utils.Equal(t, session.TargetBranch, "tgt")
		utils.Equal(t, session.ModelKey, "model-key")
		utils.Equal(t, session.Provider, "anthropic")
		utils.Equal(t, session.DocsBranch, "docs/src")
		session.ID = 9
		return nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	res, err := svc.Create(&models.GenerationSession{
		ProjectID:        3,
		SourceBranch:     " src ",
		TargetBranch:     " tgt ",
		Provider:         " anthropic ",
		ModelKey:         " model-key ",
		DocsBranch:       " docs/src ",
		MessagesJSON:     "{}",
		ChatMessagesJSON: " [] ",
	})
	utils.NilError(t, err)
	utils.Equal(t, res.ID, uint(9))
	utils.Equal(t, res.SourceBranch, "src")
	utils.Equal(t, res.TargetBranch, "tgt")
	utils.Equal(t, res.ModelKey, "model-key")
	utils.Equal(t, res.Provider, "anthropic")
	utils.Equal(t, res.DocsBranch, "docs/src")
	utils.Equal(t, res.MessagesJSON, "{}")
	utils.Equal(t, res.ChatMessagesJSON, " [] ")
}

func TestGenerationSessionService_UpdateByID_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	err := svc.UpdateByID(0, map[string]interface{}{"provider": "anthropic"})
	utils.Equal(t, err.Error(), "session ID is required")
}

func TestGenerationSessionService_UpdateByID_Delegates(t *testing.T) {
	ctx := context.Background()
	called := false
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.UpdateByIDFunc = func(id uint, updates map[string]interface{}) error {
		called = true
		utils.Equal(t, id, uint(3))
		utils.Equal(t, updates["provider"], "anthropic")
		return nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	err := svc.UpdateByID(3, map[string]interface{}{"provider": "anthropic"})
	utils.NilError(t, err)
	if !called {
		t.Fatalf("repository not called")
	}
}

func TestGenerationSessionService_DeleteByID_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	err := svc.DeleteByID(0)
	utils.Equal(t, err.Error(), "session ID is required")
}

func TestGenerationSessionService_DeleteByID_Delegates(t *testing.T) {
	ctx := context.Background()
	called := false
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.DeleteByIDFunc = func(id uint) error {
		called = true
		if id != 3 {
			return errors.New("wrong session ID")
		}
		return nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	err := svc.DeleteByID(3)
	utils.NilError(t, err)
	if !called {
		t.Fatalf("repository not called")
	}
}

func TestGenerationSessionService_DeleteAll_Delegates(t *testing.T) {
	ctx := context.Background()
	called := false
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.DeleteByProjectFunc = func(projectID uint) error {
		called = true
		utils.Equal(t, projectID, uint(11))
		return nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	err := svc.DeleteAll(11)
	utils.NilError(t, err)
	if !called {
		t.Fatalf("repository not called")
	}
}
