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

func TestGenerationSessionService_Get_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	_, err := svc.Get(1, "", "t")
	utils.Equal(t, err.Error(), "source and target branches are required")

	_, err = svc.Get(1, "s", " ")
	utils.Equal(t, err.Error(), "source and target branches are required")
}

func TestGenerationSessionService_Get_TrimsAndDelegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.GetByProjectAndBranchesFunc = func(projectID uint, s, tBranch string) (*models.GenerationSession, error) {
		utils.Equal(t, s, "src")
		utils.Equal(t, tBranch, "tgt")
		return &models.GenerationSession{ProjectID: projectID, SourceBranch: s, TargetBranch: tBranch}, nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	res, err := svc.Get(7, "  src\t", "\t tgt  ")
	utils.NilError(t, err)
	utils.Equal(t, res.ProjectID, uint(7))
	utils.Equal(t, res.SourceBranch, "src")
	utils.Equal(t, res.TargetBranch, "tgt")
}

func TestGenerationSessionService_Upsert_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	_, err := svc.Upsert(1, "", "t", "model-key", "anthropic", "docs/s", "[]", "[]")
	utils.Equal(t, err.Error(), "source and target branches are required")

	_, err = svc.Upsert(1, "s", "\n\t ", "model-key", "anthropic", "docs/s", "[]", "")
	utils.Equal(t, err.Error(), "source and target branches are required")

	_, err = svc.Upsert(1, "s", "t", "model-key", "", "docs/s", "[]", "")
	utils.Equal(t, err.Error(), "provider is required")

	_, err = svc.Upsert(1, "s", "t", "model-key", "  \t", "docs/s", "[]", "")
	utils.Equal(t, err.Error(), "provider is required")
}

func TestGenerationSessionService_Upsert_TrimsAndDelegates(t *testing.T) {
	ctx := context.Background()
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.UpsertFunc = func(projectID uint, s, tBranch, modelKey, prov, docsBranch, msg, chatJSON string) (*models.GenerationSession, error) {
		utils.Equal(t, s, "src")
		utils.Equal(t, tBranch, "tgt")
		utils.Equal(t, modelKey, "model-key")
		utils.Equal(t, prov, "anthropic")
		utils.Equal(t, docsBranch, "docs/src")
		utils.Equal(t, msg, "{}")
		utils.Equal(t, chatJSON, "[]")
		return &models.GenerationSession{
			ProjectID:        projectID,
			SourceBranch:     s,
			TargetBranch:     tBranch,
			Provider:         prov,
			ModelKey:         modelKey,
			DocsBranch:       docsBranch,
			MessagesJSON:     msg,
			ChatMessagesJSON: chatJSON,
		}, nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	res, err := svc.Upsert(9, " src ", " tgt ", " model-key ", " anthropic ", " docs/src ", "{}", " [] ")
	utils.NilError(t, err)
	utils.Equal(t, res.SourceBranch, "src")
	utils.Equal(t, res.TargetBranch, "tgt")
	utils.Equal(t, res.ModelKey, "model-key")
	utils.Equal(t, res.Provider, "anthropic")
	utils.Equal(t, res.DocsBranch, "docs/src")
	utils.Equal(t, res.MessagesJSON, "{}")
}

func TestGenerationSessionService_Delete_Validation(t *testing.T) {
	ctx := context.Background()
	svc := services.NewGenerationSessionService(&mocks.GenerationSessionRepositoryMock{})
	svc.Startup(ctx)

	err := svc.Delete(1, " ", "t")
	utils.Equal(t, err.Error(), "source and target branches are required")

	err = svc.Delete(1, "s", "\t\n")
	utils.Equal(t, err.Error(), "source and target branches are required")
}

func TestGenerationSessionService_Delete_TrimsAndDelegates(t *testing.T) {
	ctx := context.Background()
	called := false
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.DeleteByProjectAndBranchesFunc = func(projectID uint, s, tBranch string) error {
		called = true
		if s != "src" || tBranch != "tgt" {
			return errors.New("did not trim")
		}
		return nil
	}

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	err := svc.Delete(3, " src ", " tgt ")
	utils.NilError(t, err)
	if !called {
		t.Fatalf("repository not called")
	}
}

func TestGenerationSessionService_DeleteAll_Delegates(t *testing.T) {
	ctx := context.Background()
	called := false
	repo := &mocks.GenerationSessionRepositoryMock{}
	repo.DeleteByProjectFunc = func(projectID uint) error { called = true; return nil }

	svc := services.NewGenerationSessionService(repo)
	svc.Startup(ctx)

	err := svc.DeleteAll(11)
	utils.NilError(t, err)
	if !called {
		t.Fatalf("repository not called")
	}
}
