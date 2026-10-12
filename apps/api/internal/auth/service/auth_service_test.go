package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"teamflow/internal/auth"
	"teamflow/internal/auth/handler"
	"teamflow/internal/faults"

	"golang.org/x/crypto/bcrypt"
)

type fakeUsers struct {
	user     *auth.User
	password string
}

func (f *fakeUsers) Create(_ context.Context, username, email, hash string) (*auth.User, error) {
	f.password = hash
	f.user = &auth.User{ID: "7", Username: username, Email: email}
	return f.user, nil
}
func (f *fakeUsers) ByUsername(_ context.Context, username string) (*auth.User, string, error) {
	if f.user == nil || f.user.Username != username {
		return nil, "", faults.ErrNotFound
	}
	return f.user, f.password, nil
}

type fakeSessions struct {
	userID uint
	token  string
	ttl    time.Duration
	err    error
}

func (f *fakeSessions) Put(_ context.Context, id uint, token string, ttl time.Duration) error {
	f.userID, f.token, f.ttl = id, token, ttl
	return f.err
}
func (f *fakeSessions) Matches(_ context.Context, id uint, token string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.userID == id && f.token == token, nil
}

type fakeSigner struct{}

func (fakeSigner) Sign(id uint, _ string, _ time.Time) (string, error) {
	return "user:" + strconv.FormatUint(uint64(id), 10), nil
}
func (fakeSigner) Parse(token string) (*TokenClaims, error) {
	value, ok := strings.CutPrefix(token, "user:")
	id, err := strconv.ParseUint(value, 10, 64)
	if !ok || err != nil {
		return nil, errors.New("bad token")
	}
	return &TokenClaims{UserID: uint(id), Username: "user"}, nil
}

func TestRegisterStoresBcryptHash(t *testing.T) {
	users := &fakeUsers{}
	svc := NewAuthService(users, &fakeSessions{}, fakeSigner{})
	_, err := svc.Register(context.Background(), &handler.RegisterReq{Username: " user ", Email: "USER@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if users.password == "password123" {
		t.Fatal("password stored as plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(users.password), []byte("password123")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}
}

func TestLoginAndAuthenticateRequireCurrentRedisSession(t *testing.T) {
	users := &fakeUsers{user: &auth.User{ID: "7", Username: "user"}}
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	users.password = string(hash)
	sessions := &fakeSessions{}
	svc := NewAuthService(users, sessions, fakeSigner{})
	resp, err := svc.Login(context.Background(), &handler.LoginReq{Username: "user", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if sessions.ttl != 2*time.Hour {
		t.Fatalf("session TTL = %s, want 2h", sessions.ttl)
	}
	if _, err := svc.Authenticate(context.Background(), resp.Token); err != nil {
		t.Fatalf("authenticate valid token: %v", err)
	}
	sessions.token = "replaced"
	if _, err := svc.Authenticate(context.Background(), resp.Token); !errors.Is(err, faults.ErrUnauthenticated) {
		t.Fatalf("old token error = %v, want unauthenticated", err)
	}
}

func TestLoginDoesNotRevealUnknownUser(t *testing.T) {
	svc := NewAuthService(&fakeUsers{}, &fakeSessions{}, fakeSigner{})
	_, err := svc.Login(context.Background(), &handler.LoginReq{Username: "missing", Password: "wrong"})
	if !errors.Is(err, faults.ErrUnauthenticated) {
		t.Fatalf("login error = %v, want unauthenticated", err)
	}
}
