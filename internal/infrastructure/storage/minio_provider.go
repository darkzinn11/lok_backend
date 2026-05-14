package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioProvider struct {
	client *minio.Client
}

func NewMinioProvider(endpoint, accessKey, secretKey string, useSSL bool) (*MinioProvider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinioProvider{client: client}, nil
}

func (p *MinioProvider) Upload(ctx context.Context, bucket, key string, content io.Reader, contentType string) (string, error) {
	_, err := p.client.PutObject(ctx, bucket, key, content, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

func (p *MinioProvider) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := p.client.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (p *MinioProvider) GetPresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	presignedURL, err := p.client.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (p *MinioProvider) Delete(ctx context.Context, bucket, key string) error {
	return p.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

var _ domain.StorageProvider = (*MinioProvider)(nil)
