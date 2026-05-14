package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Token           string
	ExpiresAt       time.Time
	CreatedAt       time.Time
	RevokedAt       *time.Time
	ReplacedByToken string
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsActive() bool {
	return rt.RevokedAt == nil && !rt.IsExpired()
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, rt *RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error)
	UpdateRefreshToken(ctx context.Context, rt *RefreshToken) error
	RevokeTokensByUserID(ctx context.Context, userID uuid.UUID) error
}

type TokenService interface {
	GenerateTokenPair(user *User) (*TokenPair, error)
	ValidateAccessToken(token string) (*UserClaims, error)
	ValidateRefreshToken(token string) (uuid.UUID, error)
}

type UserClaims struct {
	UserID uuid.UUID
	Role   Role
}
