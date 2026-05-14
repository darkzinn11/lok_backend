package auth

import (
	"errors"
	"fmt"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTTokenManager struct {
	secretKey           string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

func NewJWTTokenManager(secretKey string, accessDuration, refreshDuration time.Duration) *JWTTokenManager {
	return &JWTTokenManager{
		secretKey:           secretKey,
		accessTokenDuration:  accessDuration,
		refreshTokenDuration: refreshDuration,
	}
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID   `json:"user_id"`
	Role     domain.Role `json:"role"`
	BranchID *uuid.UUID  `json:"branch_id,omitempty"`
}

func (m *JWTTokenManager) GenerateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	// Access Token
	accessClaims := &UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
		UserID:   user.ID,
		Role:     user.Role,
		BranchID: user.BranchID,
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(m.secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token - Secure Random string (we'll store this in DB)
	refreshToken := m.generateRandomToken()

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (m *JWTTokenManager) ValidateAccessToken(tokenString string) (*domain.UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("access token parse error: %w", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token")
	}

	return &domain.UserClaims{
		UserID: claims.UserID,
		Role:   claims.Role,
	}, nil
}

func (m *JWTTokenManager) generateRandomToken() string {
	return uuid.New().String() + uuid.New().String()
}
