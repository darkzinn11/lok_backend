package persistence

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type schemaMigration struct {
	Version string `gorm:"primaryKey;size:255"`
}

func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	migrationsDir := "migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		version := entry.Name()
		var count int64
		if err := db.Model(&schemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return fmt.Errorf("execute migration %s: %w", version, err)
			}

			return tx.Create(&schemaMigration{Version: version}).Error
		}); err != nil {
			return err
		}

		slog.Info("Applied migration", slog.String("version", version))
	}

	return nil
}
