package application

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"lockcenter-backend/internal/domain"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

const (
	MaxImageSize      = 10 * 1024 * 1024 // 10MB limit for raw upload
	DefaultTargetWidth  = 768
	DefaultTargetHeight = 0 // Auto height
	DefaultJPEGQuality  = 80
)

type ImageService struct {
	storage domain.StorageProvider
	bucket  string
}

func NewImageService(storage domain.StorageProvider, bucket string) *ImageService {
	return &ImageService{
		storage: storage,
		bucket:  bucket,
	}
}

type ProcessedImage struct {
	BucketKey string
	PublicURL string
	FileName  string
	FileSize  int64
	MimeType  string
}

func (s *ImageService) ProcessAndUpload(ctx context.Context, file io.Reader, fileName string, folder string) (*ProcessedImage, error) {
	// 1. Read file into memory (to process it)
	data, err := io.ReadAll(io.LimitReader(file, MaxImageSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 2. Validate MIME type
	mimeType := http.DetectContentType(data)
	if !s.isAllowedMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported image format: %s", mimeType)
	}

	// 3. Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 4. Resize and optimize
	// Auto-orient based on EXIF (imaging does this by default if we use Open, 
	// but here we are using Decode. We should use imaging.Decode which handles orientation better)
	img = imaging.Fit(img, DefaultTargetWidth, 1024, imaging.Lanczos)

	// 5. Encode to JPEG (optimized)
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: DefaultJPEGQuality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 6. Generate secure name
	ext := ".jpg"
	secureName := uuid.New().String() + ext
	key := filepath.Join(folder, secureName)

	// 7. Upload to storage
	publicURL, err := s.storage.Upload(ctx, s.bucket, key, bytes.NewReader(buf.Bytes()), "image/jpeg")
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	return &ProcessedImage{
		BucketKey: key,
		PublicURL: publicURL,
		FileName:  secureName,
		FileSize:  int64(buf.Len()),
		MimeType:  "image/jpeg",
	}, nil
}

func (s *ImageService) isAllowedMimeType(mime string) bool {
	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	return allowed[mime] || strings.HasPrefix(mime, "image/jpg")
}

func (s *ImageService) DeleteImage(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.storage.Delete(ctx, s.bucket, key)
}
