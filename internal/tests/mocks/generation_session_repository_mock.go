package mocks

import (
	"narrabyte/internal/models"
)

type GenerationSessionRepositoryMock struct {
	ListByProjectFunc   func(projectID uint) ([]models.GenerationSession, error)
	GetByIDFunc         func(id uint) (*models.GenerationSession, error)
	GetByDocsBranchFunc func(projectID uint, docsBranch string) (*models.GenerationSession, error)
	CreateFunc          func(session *models.GenerationSession) error
	UpdateByIDFunc      func(id uint, updates map[string]interface{}) error
	DeleteByIDFunc      func(id uint) error
	DeleteByProjectFunc func(projectID uint) error
}

func (m *GenerationSessionRepositoryMock) ListByProject(projectID uint) ([]models.GenerationSession, error) {
	if m.ListByProjectFunc != nil {
		return m.ListByProjectFunc(projectID)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) GetByID(id uint) (*models.GenerationSession, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) GetByDocsBranch(projectID uint, docsBranch string) (*models.GenerationSession, error) {
	if m.GetByDocsBranchFunc != nil {
		return m.GetByDocsBranchFunc(projectID, docsBranch)
	}
	return nil, nil
}

func (m *GenerationSessionRepositoryMock) Create(session *models.GenerationSession) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(session)
	}
	return nil
}

func (m *GenerationSessionRepositoryMock) UpdateByID(id uint, updates map[string]interface{}) error {
	if m.UpdateByIDFunc != nil {
		return m.UpdateByIDFunc(id, updates)
	}
	return nil
}

func (m *GenerationSessionRepositoryMock) DeleteByID(id uint) error {
	if m.DeleteByIDFunc != nil {
		return m.DeleteByIDFunc(id)
	}
	return nil
}

func (m *GenerationSessionRepositoryMock) DeleteByProject(projectID uint) error {
	if m.DeleteByProjectFunc != nil {
		return m.DeleteByProjectFunc(projectID)
	}
	return nil
}
