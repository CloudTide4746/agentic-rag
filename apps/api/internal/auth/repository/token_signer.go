package repository

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"teamflow/internal/auth/service"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type JWTSigner struct{ secret []byte }

type jwtClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwtlib.RegisteredClaims
}

func NewJWTSignerFromEnv() (*JWTSigner, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 characters")
	}
	return &JWTSigner{secret: []byte(secret)}, nil
}
func (s *JWTSigner) Sign(userID uint, username string, expiresAt time.Time) (string, error) {
	claims := &jwtClaims{UserID: userID, Username: username, RegisteredClaims: jwtlib.RegisteredClaims{Issuer: "agentic-rag", IssuedAt: jwtlib.NewNumericDate(time.Now().UTC()), ExpiresAt: jwtlib.NewNumericDate(expiresAt)}}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(s.secret)
}
func (s *JWTSigner) Parse(raw string) (*service.TokenClaims, error) {
	claims := &jwtClaims{}
	token, err := jwtlib.ParseWithClaims(raw, claims, func(token *jwtlib.Token) (any, error) {
		if token.Method != jwtlib.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return s.secret, nil
	}, jwtlib.WithIssuer("agentic-rag"))
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.UserID == 0 {
		return nil, errors.New("invalid token")
	}
	return &service.TokenClaims{UserID: claims.UserID, Username: claims.Username}, nil
}

var _ service.TokenSigner = (*JWTSigner)(nil)
