//go:build !prod

package database

// GetDefaultDBPath returns the database path for development mode.
// In dev mode, the database is stored in the project root for easy access and debugging.
func GetDefaultDBPath() string {
	return "narrabyte.db"
}

func IsDevelopment() bool {
	return true
}
