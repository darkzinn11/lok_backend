package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"lockcenter-backend/internal/domain"
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

func (s *LocalStorage) Upload(ctx context.Context, bucket, key string, content io.Reader, contentType string) (string, error) {
	fullPath := filepath.Join(s.basePath, bucket, key)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// For local storage, the public URL is usually served via a static route
	return fmt.Sprintf("/uploads/%s/%s", bucket, key), nil
}

func (s *LocalStorage) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("/uploads/%s/%s", bucket, key), nil
}

func (s *LocalStorage) GetPresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return "", fmt.Errorf("not implemented for local storage")
}

func (s *LocalStorage) Delete(ctx context.Context, bucket, key string) error {
	fullPath := filepath.Join(s.basePath, bucket, key)
	return os.Remove(fullPath)
}

var _ domain.StorageProvider = (*LocalStorage)(nil)
