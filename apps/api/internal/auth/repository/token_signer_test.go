package repository

import (
	"strings"
	"testing"
	"time"
)

func TestJWTSignerValidatesSignatureAndExpiry(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("x", 40))
	signer, err := NewJWTSignerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.Sign(5, "user", time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := signer.Parse(token)
	if err != nil {
		t.Fatalf("parse valid token: %v", err)
	}
	if claims.UserID != 5 || claims.Username != "user" {
		t.Fatalf("claims = %#v", claims)
	}
	if _, err := signer.Parse(token + "tampered"); err == nil {
		t.Fatal("tampered token was accepted")
	}
	expired, err := signer.Sign(5, "user", time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Parse(expired); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestJWTSignerRejectsShortSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "short")
	if _, err := NewJWTSignerFromEnv(); err == nil {
		t.Fatal("short JWT secret was accepted")
	}
}
