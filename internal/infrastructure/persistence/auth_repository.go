package persistence

import (
	"context"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormAuthRepository struct {
	db *gorm.DB
}

func NewGormAuthRepository(db *gorm.DB) *GormAuthRepository {
	return &GormAuthRepository{db: db}
}

func (r *GormAuthRepository) CreateRefreshToken(ctx context.Context, rt *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(rt).Error
}

func (r *GormAuthRepository) GetRefreshToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&rt).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *GormAuthRepository) UpdateRefreshToken(ctx context.Context, rt *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Save(rt).Error
}

func (r *GormAuthRepository) RevokeTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}
