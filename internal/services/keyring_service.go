package services

import (
	"errors"
	"runtime"

	"github.com/99designs/keyring"
)

func GetOS() string {
	return runtime.GOOS
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

func StoreApiKey(r keyring.Keyring, provider string, apiKey []byte) error {
	if r == nil {
		return errors.New("Keyring is not initialized")
	}
	if len(apiKey) == 0 {
		return errors.New("API key is empty")
	}
	if provider == "" {
		return errors.New("Provider is required")
	}

	item := keyring.Item{
		//Important : Attribute "Key" is used to retrieve the item later.
		//If the prefex or suffix is changed, be sure to update it in the GetApiKey and DeleteApiKey functions.
		Key:         "narrabyte:" + provider + ":apikey",
		Data:        apiKey,
		Label:       provider + "API key",
		Description: "API key for " + provider + "used by Narrabyte",
	}
	return r.Set(item)
}

func GetApiKey(r keyring.Keyring, provider string) (string, error) {
	if r == nil {
		return "", errors.New("Keyring is not initialized")
	}
	if provider == "" {
		return "", errors.New("Provider is required")
	}
	//Be sure to match the key format used in StoreApiKey
	item, err := r.Get("narrabyte:" + provider + ":apiKey")
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

func DeleteApiKey(r keyring.Keyring, provider string) error {
	if r == nil {
		return errors.New("Keyring is not initialized")
	}
	if provider == "" {
		return errors.New("Provider is required")
	}

	return r.Remove("narrabyte:" + provider + ":apiKey")
}
