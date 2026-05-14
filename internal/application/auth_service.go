package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/auth"
	"lockcenter-backend/internal/infrastructure/security"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo       domain.UserRepository
	authRepo       domain.AuthRepository
	tokenManager   *auth.JWTTokenManager
	passwordHasher security.PasswordHasher
}

type UserProfileOutput struct {
	ID        uuid.UUID         `json:"id"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Phone     string            `json:"phone"`
	PhotoURL  string            `json:"photo_url,omitempty"`
	Role      domain.Role       `json:"role"`
	Status    domain.UserStatus `json:"status"`
	BranchID           *uuid.UUID        `json:"branch_id,omitempty"`
	MustChangePassword bool              `json:"must_change_password"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func NewAuthService(
	userRepo domain.UserRepository,
	authRepo domain.AuthRepository,
	tokenManager *auth.JWTTokenManager,
	passwordHasher security.PasswordHasher,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		authRepo:       authRepo,
		tokenManager:   tokenManager,
		passwordHasher: passwordHasher,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.TokenPair, error) {
	slog.Debug("Processing login", slog.String("email", email))
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		slog.Error("User not found by email", slog.String("email", email), slog.Any("error", err))
		return nil, errors.New("invalid email or password")
	}

	if err := s.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		slog.Warn("Password comparison failed", slog.String("email", email), slog.Any("error", err))
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "" && user.Status != domain.UserStatusActive {
		return nil, errors.New("invalid email or password")
	}

	tokenPair, err := s.tokenManager.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Persist refresh token
	refreshToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.authRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return tokenPair, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*domain.TokenPair, error) {
	rt, err := s.authRepo.GetRefreshToken(ctx, refreshTokenStr)
	if err != nil || !rt.IsActive() {
		return nil, errors.New("invalid or expired refresh token")
	}

	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Rotate tokens
	tokenPair, err := s.tokenManager.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Revoke current token
	now := time.Now()
	rt.RevokedAt = &now
	rt.ReplacedByToken = tokenPair.RefreshToken
	if err := s.authRepo.UpdateRefreshToken(ctx, rt); err != nil {
		return nil, err
	}

	// Persist NEW refresh token
	newRt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.authRepo.CreateRefreshToken(ctx, newRt); err != nil {
		return nil, err
	}

	return tokenPair, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshTokenStr string) error {
	rt, err := s.authRepo.GetRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		return nil // Already invalid or gone
	}

	now := time.Now()
	rt.RevokedAt = &now
	return s.authRepo.UpdateRefreshToken(ctx, rt)
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfileOutput, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return &UserProfileOutput{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		PhotoURL:  user.PhotoURL,
		Role:      user.Role,
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		BranchID:           user.BranchID,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	hash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	user.MustChangePassword = false
	user.UpdatedAt = time.Now().UTC()

	return s.userRepo.Update(ctx, user)
}
