package domain

import (
	"context"
	"io"
	"time"
)

type StorageProvider interface {
	Upload(ctx context.Context, bucket, key string, content io.Reader, contentType string) (string, error)
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	GetPresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, bucket, key string) error
}
