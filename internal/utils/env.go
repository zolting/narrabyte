package utils

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

func LoadEnv() error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	envPath := filepath.Join(root, ".env")
	return godotenv.Load(envPath)
}
