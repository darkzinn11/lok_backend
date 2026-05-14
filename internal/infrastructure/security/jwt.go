package security

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID         uuid.UUID `json:"user_id"`
	Role           string    `json:"role"`
	MunicipalityID uuid.UUID `json:"municipality_id"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey []byte
	issuer    string
}

func NewJWTService() *JWTService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret_key_change_me"
	}
	return &JWTService{
		secretKey: []byte(secret),
		issuer:    "lockcenter-api",
	}
}

func (s *JWTService) GenerateToken(userID uuid.UUID, role string, municipalityID uuid.UUID) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:         userID,
		Role:           role,
		MunicipalityID: municipalityID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
