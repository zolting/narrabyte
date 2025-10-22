package services

import (
	"context"
	"encoding/json"
	"fmt"
	"narrabyte/internal/assets"
	"narrabyte/internal/models"
	"narrabyte/internal/repositories"
	"sort"
	"strings"
	"sync"
)

type ModelConfigService interface {
	Startup(ctx context.Context) error
	ListModelGroups() ([]models.LLMModelGroup, error)
	SetModelEnabled(modelKey string, enabled bool) (*models.LLMModel, error)
	SetProviderEnabled(provider string, enabled bool) ([]models.LLMModel, error)
	GetModel(modelKey string) (*models.LLMModel, error)
}

type modelConfigService struct {
	repo repositories.ModelSettingRepository
	ctx  context.Context

	mu            sync.RWMutex
	providerOrder []string
	providerNames map[string]string
	models        map[string]*catalogModel
	settings      map[string]bool
}

type catalogModel struct {
	Key         string
	ProviderID  string
	Provider    string
	DisplayName string
	APIName     string

	ReasoningEffort string
	Thinking        *bool
}

type rawModelFile struct {
	Providers []rawProvider `json:"providers"`
}

type rawProvider struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"displayName"`
	Models      []rawModel `json:"models"`
}

type rawModel struct {
	DisplayName     string `json:"displayName"`
	APIName         string `json:"apiName"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
	Thinking        *bool  `json:"thinking,omitempty"`
}

func NewModelConfigService(repo repositories.ModelSettingRepository) ModelConfigService {
	return &modelConfigService{
		repo:          repo,
		models:        make(map[string]*catalogModel),
		settings:      make(map[string]bool),
		providerNames: make(map[string]string),
		mu:            sync.RWMutex{},
	}
}

func (s *modelConfigService) Startup(ctx context.Context) error {
	s.ctx = ctx

	var parsed rawModelFile
	if err := json.Unmarshal(assets.ModelsData, &parsed); err != nil {
		return fmt.Errorf("parse models asset: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.providerOrder = make([]string, 0, len(parsed.Providers))
	for _, provider := range parsed.Providers {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" {
			continue
		}
		providerName := strings.TrimSpace(provider.DisplayName)
		s.providerNames[providerID] = providerName
		s.providerOrder = append(s.providerOrder, providerID)
		for _, mdl := range provider.Models {
			key := computeModelKey(providerID, mdl)
			s.models[key] = &catalogModel{
				Key:             key,
				ProviderID:      providerID,
				Provider:        providerName,
				DisplayName:     strings.TrimSpace(mdl.DisplayName),
				APIName:         strings.TrimSpace(mdl.APIName),
				ReasoningEffort: strings.TrimSpace(mdl.ReasoningEffort),
				Thinking:        mdl.Thinking,
			}
		}
	}

	// Load existing settings and seed defaults
	existing, err := s.repo.List()
	if err != nil {
		return fmt.Errorf("load model settings: %w", err)
	}
	for _, setting := range existing {
		s.settings[setting.ModelKey] = setting.Enabled
	}
	for key, def := range s.models {
		if _, ok := s.settings[key]; !ok {
			if _, err := s.repo.Upsert(key, def.ProviderID, true); err != nil {
				return fmt.Errorf("seed model setting for %s: %w", key, err)
			}
			s.settings[key] = true
		}
	}

	return nil
}

func (s *modelConfigService) ListModelGroups() ([]models.LLMModelGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := make([]models.LLMModelGroup, 0, len(s.providerOrder))
	for _, providerID := range s.providerOrder {
		group := models.LLMModelGroup{
			ProviderID:   providerID,
			ProviderName: s.providerName(providerID),
		}
		var modelsForProvider []models.LLMModel
		for _, mdl := range s.models {
			if mdl.ProviderID != providerID {
				continue
			}
			modelsForProvider = append(modelsForProvider, s.toLLMModel(mdl))
		}
		sort.SliceStable(modelsForProvider, func(i, j int) bool {
			return strings.ToLower(modelsForProvider[i].DisplayName) < strings.ToLower(modelsForProvider[j].DisplayName)
		})
		group.Models = modelsForProvider
		groups = append(groups, group)
	}
	return groups, nil
}

func (s *modelConfigService) SetModelEnabled(modelKey string, enabled bool) (*models.LLMModel, error) {
	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return nil, fmt.Errorf("model key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	catalog, ok := s.models[modelKey]
	if !ok {
		return nil, fmt.Errorf("model %s not found", modelKey)
	}

	if _, err := s.repo.Upsert(modelKey, catalog.ProviderID, enabled); err != nil {
		return nil, err
	}
	s.settings[modelKey] = enabled
	model := s.toLLMModel(catalog)
	return &model, nil
}

func (s *modelConfigService) SetProviderEnabled(provider string, enabled bool) ([]models.LLMModel, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.repo.SetProviderEnabled(provider, enabled); err != nil {
		return nil, err
	}

	updated := make([]models.LLMModel, 0)
	for _, mdl := range s.models {
		if mdl.ProviderID != provider {
			continue
		}
		s.settings[mdl.Key] = enabled
		updated = append(updated, s.toLLMModel(mdl))
	}
	sort.SliceStable(updated, func(i, j int) bool {
		return strings.ToLower(updated[i].DisplayName) < strings.ToLower(updated[j].DisplayName)
	})
	return updated, nil
}

func (s *modelConfigService) GetModel(modelKey string) (*models.LLMModel, error) {
	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return nil, fmt.Errorf("model key is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	catalog, ok := s.models[modelKey]
	if !ok {
		return nil, fmt.Errorf("model %s not found", modelKey)
	}
	model := s.toLLMModel(catalog)
	return &model, nil
}

func (s *modelConfigService) providerName(providerID string) string {
	if name, ok := s.providerNames[providerID]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	return providerID
}

func (s *modelConfigService) toLLMModel(mdl *catalogModel) models.LLMModel {
	enabled := s.settings[mdl.Key]
	return models.LLMModel{
		Key:             mdl.Key,
		DisplayName:     mdl.DisplayName,
		APIName:         mdl.APIName,
		ProviderID:      mdl.ProviderID,
		ProviderName:    mdl.Provider,
		ReasoningEffort: mdl.ReasoningEffort,
		Thinking:        mdl.Thinking,
		Enabled:         enabled,
	}
}

func computeModelKey(providerID string, mdl rawModel) string {
	apiName := strings.TrimSpace(mdl.APIName)
	parts := []string{strings.TrimSpace(providerID), apiName}

	var attrs []string
	if re := strings.TrimSpace(mdl.ReasoningEffort); re != "" {
		attrs = append(attrs, "reasoning="+re)
	}
	if mdl.Thinking != nil {
		attrs = append(attrs, fmt.Sprintf("thinking=%t", *mdl.Thinking))
	}
	if len(attrs) > 0 {
		sort.Strings(attrs)
		parts = append(parts, strings.Join(attrs, ","))
	}
	return strings.Join(parts, "|")
}
