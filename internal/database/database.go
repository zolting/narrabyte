package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"narrabyte/internal/models"
)

// Config holds DB configuration
type Config struct {
	Path     string
	LogLevel logger.LogLevel
}

// Init opens a SQLite DB and runs migrations
func Init(cfg Config) (*gorm.DB, error) {
	if cfg.LogLevel == 0 {
		cfg.LogLevel = logger.Warn
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON", cfg.Path)

	gormLogger := logger.New(
		log.New(loggerWriter{}, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  cfg.LogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Configure connection pool for SQLite to prevent "database is locked" errors
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// migrate runs all automigrations. Keep the model list in one place.
func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// loggerWriter satisfies io.Writer for GORM logger but delegates to std log.Printf
type loggerWriter struct{}

func (loggerWriter) Write(p []byte) (int, error) {
	log.Printf("%s", p)
	return len(p), nil
}
