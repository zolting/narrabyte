package services

import (
	"errors"
	"runtime"

	"github.com/99designs/keyring"
)

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
		//Panic ou erreur?
		panic("failed to open keyring: " + err.Error())
	}
	s.ring = r
}

func OpenKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: "narrabyte",
		AllowedBackends: []keyring.BackendType{
			keyring.WinCredBackend,       //For Windows
			keyring.KeychainBackend,      //For macOS
			keyring.SecretServiceBackend, //For Linux
			//We can allow other backends if these fail, such as an encrypted file.
			//Unemplemented for now.
		},
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

	item := keyring.Item{
		//Important : Attribute "Key" is used to retrieve the item later.
		//If the prefex or suffix is changed, be sure to update it in the GetApiKey and DeleteApiKey functions.
		Key:         "narrabyte:" + provider + ":apikey",
		Data:        apiKey,
		Label:       provider + "API key",
		Description: "API key for " + provider + "used by Narrabyte",
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
	//Be sure to match the key format used in StoreApiKey
	item, err := r.Get("narrabyte:" + provider + ":apiKey")
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

	return s.ring.Remove("narrabyte:" + provider + ":apiKey")
}
