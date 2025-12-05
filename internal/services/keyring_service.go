package services

import (
	"errors"
	"runtime"
	"strings"

	"github.com/99designs/keyring"
)

const prefix = "narrabyte:"
const suffix = ":apikey"

func GetOS() string {
	return runtime.GOOS
}

type KeyringService struct {
	ring keyring.Keyring
}

func NewKeyringService() *KeyringService {
	return &KeyringService{}
}

func (s *KeyringService) Startup() {
	r, err := OpenKeyring()
	if err != nil {
		panic("failed to open keyring: " + err.Error())
	}
	s.ring = r
}

func OpenKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: "narrabyte",
	})
}

func (s *KeyringService) StoreApiKey(provider string, apiKey []byte) error {
	if s.ring == nil {
		return errors.New("keyring is not initialized")
	}
	if len(apiKey) == 0 {
		return errors.New("API key is empty")
	}
	if provider == "" {
		return errors.New("provider is required")
	}

	key, err := s.GetApiKey(provider)
	if err == nil && key != "" {
		s.DeleteApiKey(provider)
	}

	item := keyring.Item{
		Key:         prefix + provider + suffix,
		Data:        apiKey,
		Label:       provider + "API key",
		Description: "API key for " + provider + " used by Narrabyte",
	}
	return s.ring.Set(item)
}

func (s *KeyringService) GetApiKey(provider string) (string, error) {
	if s.ring == nil {
		return "", errors.New("keyring is not initialized")
	}
	if provider == "" {
		return "", errors.New("provider is required")
	}
	// Be sure to match the key format used in StoreApiKey
	item, err := s.ring.Get(prefix + provider + suffix)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func (s *KeyringService) DeleteApiKey(provider string) error {
	if s.ring == nil {
		return errors.New("keyring is not initialized")
	}
	if provider == "" {
		return errors.New("provider is required")
	}

	return s.ring.Remove(prefix + provider + suffix)
}

func (s *KeyringService) ListApiKeys() ([]map[string]string, error) {
	if s.ring == nil {
		return nil, errors.New("keyring is not initialized")
	}

	items, err := s.ring.Keys()
	if err != nil {
		return nil, err
	}

	var results []map[string]string
	for _, key := range items {
		// Only include narrabyte keys
		if !strings.HasPrefix(key, "narrabyte:") {
			continue
		}

		item, err := s.ring.Get(key)
		if err != nil {
			continue // skip if retrieval fails
		}

		results = append(results, map[string]string{
			"provider":    strings.TrimPrefix(strings.TrimSuffix(key, ":apikey"), "narrabyte:"),
			"label":       item.Label,
			"description": item.Description,
		})
	}
	return results, nil
}
