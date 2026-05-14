package persistence

import (
	"fmt"
	"log/slog"

	"lockcenter-backend/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB represents the database connection
type DB struct {
	Gorm *gorm.DB
}

// NewDatabase initializes a new GORM connection
func NewDatabase(cfg *config.Config) (*DB, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBPort,
			cfg.DBSSLMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	slog.Info("Connected to database successfully")

	if err := RunMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &DB{Gorm: db}, nil
}
