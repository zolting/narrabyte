package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
)

const serviceName = "narrabyte"

func GetOS() string {
	return runtime.GOOS
}

type KeyringService struct {
}

func NewKeyringService() *KeyringService {
	return &KeyringService{}
}

func (s *KeyringService) StoreApiKey(provider string, apiKey []byte) error {
	if len(apiKey) == 0 {
		return errors.New("API key is empty")
	}
	if provider == "" {
		return errors.New("provider is required")
	}

	err := keyring.Set(serviceName, provider, string(apiKey))
	if err != nil {
		return err
	}

	return s.addProvider(provider)
}

func (s *KeyringService) GetApiKey(provider string) (string, error) {
	if provider == "" {
		return "", errors.New("provider is required")
	}
	return keyring.Get(serviceName, provider)
}

func (s *KeyringService) DeleteApiKey(provider string) error {
	if provider == "" {
		return errors.New("provider is required")
	}

	err := keyring.Delete(serviceName, provider)
	if err != nil {
		return err
	}

	return s.removeProvider(provider)
}

func (s *KeyringService) ListApiKeys() ([]map[string]string, error) {
	providers, err := s.loadProviders()
	if err != nil {
		return nil, err
	}

	var results []map[string]string
	for _, provider := range providers {
		_, err := keyring.Get(serviceName, provider)
		if err != nil {
			continue
		}

		results = append(results, map[string]string{
			"provider":    provider,
			"label":       provider + " API key",
			"description": "API key for " + provider + " used by Narrabyte",
		})
	}
	return results, nil
}

func (s *KeyringService) getProvidersConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "narrabyte")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "providers.json"), nil
}

func (s *KeyringService) loadProviders() ([]string, error) {
	path, err := s.getProvidersConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var providers []string
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *KeyringService) saveProviders(providers []string) error {
	path, err := s.getProvidersConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (s *KeyringService) addProvider(provider string) error {
	providers, err := s.loadProviders()
	if err != nil {
		return err
	}

	for _, p := range providers {
		if p == provider {
			return nil
		}
	}

	providers = append(providers, provider)
	return s.saveProviders(providers)
}

func (s *KeyringService) removeProvider(provider string) error {
	providers, err := s.loadProviders()
	if err != nil {
		return err
	}

	var newProviders []string
	for _, p := range providers {
		if p != provider {
			newProviders = append(newProviders, p)
		}
	}

	return s.saveProviders(newProviders)
}
