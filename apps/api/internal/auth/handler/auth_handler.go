package handler

import (
	"context"

	"teamflow/internal/auth"
)

type RegisterReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type AuthResp struct {
	Token string    `json:"token"`
	User  auth.User `json:"user"`
}

type AuthHandler interface {
	Register(ctx context.Context, req *RegisterReq) (*auth.User, error)
	Login(ctx context.Context, req *LoginReq) (*AuthResp, error)
	Authenticate(ctx context.Context, token string) (*auth.Identity, error)
}
