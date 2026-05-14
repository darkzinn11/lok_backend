package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/presentation/http/middleware"
)

type AuthHandler struct {
	*BaseHandler
	authService  *application.AuthService
	isProduction bool
}

func NewAuthHandler(authService *application.AuthService, isProd bool) *AuthHandler {
	return &AuthHandler{
		BaseHandler:  NewBaseHandler(),
		authService:  authService,
		isProduction: isProd,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8,max=72"`
	}

	if err := h.Decode(r, &req); err != nil {
		slog.Error("Failed to decode login request", slog.Any("error", err))
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Debug("Login attempt", slog.String("email", req.Email))

	tokenPair, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		slog.Warn("Login failed", slog.String("email", req.Email), slog.Any("error", err))
		h.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// Set Refresh Token in HTTP-Only Cookie
	h.setRefreshTokenCookie(w, tokenPair.RefreshToken)

	h.Success(w, map[string]string{
		"access_token": tokenPair.AccessToken,
	}, http.StatusOK)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		h.Error(w, "refresh token missing", http.StatusUnauthorized)
		return
	}

	tokenPair, err := h.authService.RefreshToken(r.Context(), cookie.Value)
	if err != nil {
		h.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// Rotate cookie
	h.setRefreshTokenCookie(w, tokenPair.RefreshToken)

	h.Success(w, map[string]string{
		"access_token": tokenPair.AccessToken,
	}, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		_ = h.authService.Logout(r.Context(), cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})

	h.Success(w, map[string]string{"message": "logged out successfully"}, http.StatusOK)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.authService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	h.Success(w, user, http.StatusOK)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
	}

	if err := h.Decode(r, &req); err != nil {
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.authService.ChangePassword(r.Context(), claims.UserID, req.NewPassword); err != nil {
		h.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Success(w, map[string]string{"message": "password changed successfully"}, http.StatusOK)
}

func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
	})
}
