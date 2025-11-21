//go:build prod

package database

import (
	"log"
	"os"
	"path/filepath"
)

// GetDefaultDBPath returns the database path for production mode.
// In production, the database is stored in the user's config directory.
func GetDefaultDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Warning: Failed to get user config dir: %v. Using fallback.", err)
		return "narrabyte.db"
	}

	appDir := filepath.Join(configDir, "narrabyte")

	err = os.MkdirAll(appDir, 0755)
	if err != nil {
		log.Printf("Warning: Failed to create app config dir: %v. Using fallback.", err)
		return "narrabyte.db"
	}

	dbPath := filepath.Join(appDir, "narrabyte.db")

	return dbPath
}

func IsDevelopment() bool {
	return false
}
