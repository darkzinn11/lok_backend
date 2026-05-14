package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	// Create base path if not exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		os.MkdirAll(basePath, 0755)
	}
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) Upload(ctx context.Context, filename string, data []byte) (string, error) {
	// Generate unique filename to avoid collisions
	ext := filepath.Ext(filename)
	newFilename := uuid.New().String() + ext
	fullPath := filepath.Join(s.basePath, newFilename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return fullPath, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(path)
}
