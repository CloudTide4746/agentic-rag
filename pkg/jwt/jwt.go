package jwt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

const (
	TokenExpireDuration        = 2 * time.Hour
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

var (
	secretMu  sync.RWMutex
	jwtSecret []byte
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwtlib.RegisteredClaims
}

func Configure(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("JWT secret is required")
	}
	secretMu.Lock()
	jwtSecret = []byte(secret)
	secretMu.Unlock()
	return nil
}

func GenerateToken(userID uint, username, role string) (string, error) {
	secret, err := signingSecret()
	if err != nil {
		return "", err
	}
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwtlib.RegisteredClaims{
			IssuedAt:  jwtlib.NewNumericDate(time.Now()),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(TokenExpireDuration)),
			Issuer:    "Teamflow",
		},
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseToken(tokenString string) (*Claims, error) {
	secret, err := signingSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwtlib.Token) (interface{}, error) {
			if token.Method != jwtlib.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
			}
			return secret, nil
		},
		jwtlib.WithIssuer("Teamflow"),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func RefreshToken(tokenString string) (string, error) {
	secret, err := signingSecret()
	if err != nil {
		return "", err
	}
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwtlib.Token) (interface{}, error) {
			if token.Method != jwtlib.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
			}
			return secret, nil
		},
		jwtlib.WithoutClaimsValidation(),
	)
	if err != nil {
		return "", fmt.Errorf("parse refresh token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.ExpiresAt == nil {
		return "", errors.New("invalid token claims")
	}
	if time.Since(claims.ExpiresAt.Time) > RefreshTokenExpireDuration {
		return "", errors.New("refresh window expired")
	}
	return GenerateToken(claims.UserID, claims.Username, claims.Role)
}

func signingSecret() ([]byte, error) {
	secretMu.RLock()
	defer secretMu.RUnlock()
	if len(jwtSecret) == 0 {
		return nil, errors.New("JWT is not configured")
	}
	secret := make([]byte, len(jwtSecret))
	copy(secret, jwtSecret)
	return secret, nil
}
