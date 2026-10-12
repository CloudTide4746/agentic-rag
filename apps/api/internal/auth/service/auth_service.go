package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"teamflow/internal/auth"
	"teamflow/internal/auth/handler"
	"teamflow/internal/faults"

	"golang.org/x/crypto/bcrypt"
)

const TokenTTL = 2 * time.Hour

type UserRepository interface {
	Create(ctx context.Context, username, email, passwordHash string) (*auth.User, error)
	ByUsername(ctx context.Context, username string) (*auth.User, string, error)
}
type SessionStore interface {
	Put(ctx context.Context, userID uint, token string, ttl time.Duration) error
	Matches(ctx context.Context, userID uint, token string) (bool, error)
}
type TokenSigner interface {
	Sign(userID uint, username string, expiresAt time.Time) (string, error)
	Parse(token string) (*TokenClaims, error)
}
type TokenClaims struct {
	UserID   uint
	Username string
}

type AuthService struct {
	users    UserRepository
	sessions SessionStore
	tokens   TokenSigner
}

var _ handler.AuthHandler = (*AuthService)(nil)

func NewAuthService(users UserRepository, sessions SessionStore, tokens TokenSigner) *AuthService {
	return &AuthService{users: users, sessions: sessions, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, req *handler.RegisterReq) (*auth.User, error) {
	if req == nil {
		return nil, fmt.Errorf("request required: %w", faults.ErrValidation)
	}
	username, email := strings.TrimSpace(req.Username), strings.TrimSpace(strings.ToLower(req.Email))
	parsedEmail, emailErr := mail.ParseAddress(email)
	if len(username) < 2 || utf8.RuneCountInString(username) > 50 || emailErr != nil || parsedEmail.Address != email || utf8.RuneCountInString(email) > 100 || utf8.RuneCountInString(req.Password) < 8 || len(req.Password) > 72 {
		return nil, fmt.Errorf("username, email or password is invalid: %w", faults.ErrValidation)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return s.users.Create(ctx, username, email, string(hash))
}

func (s *AuthService) Login(ctx context.Context, req *handler.LoginReq) (*handler.AuthResp, error) {
	if req == nil || strings.TrimSpace(req.Username) == "" || req.Password == "" {
		return nil, fmt.Errorf("username and password are required: %w", faults.ErrValidation)
	}
	user, hash, err := s.users.ByUsername(ctx, strings.TrimSpace(req.Username))
	if err != nil {
		if errors.Is(err, faults.ErrNotFound) {
			return nil, fmt.Errorf("invalid username or password: %w", faults.ErrUnauthenticated)
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		return nil, fmt.Errorf("invalid username or password: %w", faults.ErrUnauthenticated)
	}
	expiresAt := time.Now().UTC().Add(TokenTTL)
	userID, err := strconv.ParseUint(user.ID, 10, 64)
	if err != nil || userID == 0 {
		return nil, fmt.Errorf("invalid user identity: %w", faults.ErrConflict)
	}
	token, err := s.tokens.Sign(uint(userID), user.Username, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	if err := s.sessions.Put(ctx, uint(userID), token, TokenTTL); err != nil {
		return nil, fmt.Errorf("save login session: %w", err)
	}
	return &handler.AuthResp{Token: token, User: *user}, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (*auth.Identity, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", faults.ErrUnauthenticated)
	}
	active, err := s.sessions.Matches(ctx, claims.UserID, token)
	if err != nil {
		return nil, fmt.Errorf("check login session: %w", err)
	}
	if !active {
		return nil, faults.ErrUnauthenticated
	}
	return &auth.Identity{UserID: claims.UserID, Username: claims.Username}, nil
}
