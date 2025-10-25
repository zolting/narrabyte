package mocks

import (
	"narrabyte/internal/models"
)

type GenerationSessionRepositoryMock struct {
	ListByProjectFunc              func(projectID uint) ([]models.GenerationSession, error)
	GetByProjectAndBranchesFunc    func(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error)
	UpsertFunc                     func(projectID uint, sourceBranch, targetBranch, modelKey, provider, messagesJSON string) (*models.GenerationSession, error)
	DeleteByProjectFunc            func(projectID uint) error
	DeleteByProjectAndBranchesFunc func(projectID uint, sourceBranch, targetBranch string) error
}

func (m *GenerationSessionRepositoryMock) ListByProject(projectID uint) ([]models.GenerationSession, error) {
	if m.ListByProjectFunc != nil {
		return m.ListByProjectFunc(projectID)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) GetByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error) {
	if m.GetByProjectAndBranchesFunc != nil {
		return m.GetByProjectAndBranchesFunc(projectID, sourceBranch, targetBranch)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) Upsert(projectID uint, sourceBranch, targetBranch, modelKey, provider, messagesJSON string) (*models.GenerationSession, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(projectID, sourceBranch, targetBranch, modelKey, provider, messagesJSON)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) DeleteByProject(projectID uint) error {
	if m.DeleteByProjectFunc != nil {
		return m.DeleteByProjectFunc(projectID)
	}
	return nil
}

func (m *GenerationSessionRepositoryMock) DeleteByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) error {
	if m.DeleteByProjectAndBranchesFunc != nil {
		return m.DeleteByProjectAndBranchesFunc(projectID, sourceBranch, targetBranch)
	}
	return nil
}
