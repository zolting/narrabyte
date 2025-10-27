package repositories

import (
	"errors"
	"fmt"
	"narrabyte/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GenerationSessionRepository interface {
	ListByProject(projectID uint) ([]models.GenerationSession, error)
	GetByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error)
	Upsert(projectID uint, sourceBranch, targetBranch, modelKey, provider, messagesJSON string) (*models.GenerationSession, error)
	DeleteByProject(projectID uint) error
	DeleteByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) error
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

func (r *generationSessionRepository) GetByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) (*models.GenerationSession, error) {
	var sess models.GenerationSession
	res := r.db.Where("project_id = ? AND source_branch = ? AND target_branch = ?", projectID, sourceBranch, targetBranch).Take(&sess)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, res.Error
	}
	return &sess, nil
}

func (r *generationSessionRepository) Upsert(projectID uint, sourceBranch, targetBranch, modelKey, provider, messagesJSON string) (*models.GenerationSession, error) {
	if projectID == 0 {
		return nil, fmt.Errorf("projectID is required")
	}
	if sourceBranch == "" || targetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	sess := models.GenerationSession{
		ProjectID:    projectID,
		SourceBranch: sourceBranch,
		TargetBranch: targetBranch,
		Provider:     provider,
		ModelKey:     modelKey,
		MessagesJSON: messagesJSON,
	}
	// Upsert on composite unique index
	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source_branch"}, {Name: "target_branch"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider", "model_key", "messages_json", "updated_at"}),
	}).Create(&sess).Error; err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *generationSessionRepository) DeleteByProject(projectID uint) error {
	return r.db.Where("project_id = ?", projectID).Delete(&models.GenerationSession{}).Error
}

func (r *generationSessionRepository) DeleteByProjectAndBranches(projectID uint, sourceBranch, targetBranch string) error {
	return r.db.Where("project_id = ? AND source_branch = ? AND target_branch = ?", projectID, sourceBranch, targetBranch).Delete(&models.GenerationSession{}).Error
}
