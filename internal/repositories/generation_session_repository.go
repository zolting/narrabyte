package repositories

import (
	"errors"
	"fmt"
	"narrabyte/internal/models"
	"time"

	"gorm.io/gorm"
)

type GenerationSessionRepository interface {
	ListByProject(projectID uint) ([]models.GenerationSession, error)
	GetByID(id uint) (*models.GenerationSession, error)
	GetByDocsBranch(projectID uint, docsBranch string) (*models.GenerationSession, error)
	Create(session *models.GenerationSession) error
	UpdateByID(id uint, updates map[string]interface{}) error
	DeleteByID(id uint) error
	DeleteByProject(projectID uint) error
}

type generationSessionRepository struct {
	db *gorm.DB
}

func NewGenerationSessionRepository(db *gorm.DB) GenerationSessionRepository {
	return &generationSessionRepository{db: db}
}

func (r *generationSessionRepository) ListByProject(projectID uint) ([]models.GenerationSession, error) {
	var sessions []models.GenerationSession
	res := r.db.Where("project_id = ?", projectID).Order("updated_at desc").Find(&sessions)
	if res.Error != nil {
		return nil, res.Error
	}
	return sessions, nil
}

func (r *generationSessionRepository) GetByID(id uint) (*models.GenerationSession, error) {
	var sess models.GenerationSession
	res := r.db.First(&sess, id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, res.Error
	}
	return &sess, nil
}

func (r *generationSessionRepository) GetByDocsBranch(projectID uint, docsBranch string) (*models.GenerationSession, error) {
	var sess models.GenerationSession
	res := r.db.Where("project_id = ? AND docs_branch = ?", projectID, docsBranch).Take(&sess)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, res.Error
	}
	return &sess, nil
}

func (r *generationSessionRepository) Create(session *models.GenerationSession) error {
	if session.ProjectID == 0 {
		return fmt.Errorf("projectID is required")
	}
	if session.SourceBranch == "" || session.TargetBranch == "" {
		return fmt.Errorf("source and target branches are required")
	}
	if session.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if session.DocsBranch == "" {
		return fmt.Errorf("docsBranch is required")
	}
	return r.db.Create(session).Error
}

func (r *generationSessionRepository) UpdateByID(id uint, updates map[string]interface{}) error {
	if id == 0 {
		return fmt.Errorf("session ID is required")
	}
	updates["updated_at"] = time.Now()
	return r.db.Model(&models.GenerationSession{}).Where("id = ?", id).Updates(updates).Error
}

func (r *generationSessionRepository) DeleteByID(id uint) error {
	return r.db.Delete(&models.GenerationSession{}, id).Error
}

func (r *generationSessionRepository) DeleteByProject(projectID uint) error {
	return r.db.Where("project_id = ?", projectID).Delete(&models.GenerationSession{}).Error
}
