package auth

import (
	"context"

	authapi "github.com/jannfis/argocd-application-agent/pkg/api/grpc/auth"
)

type AuthServer struct {
	authapi.UnimplementedAuthenticationServer
}

func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

func (s *AuthServer) Authenticate(ctx context.Context, a *authapi.AuthRequest) (*authapi.AuthResponse, error) {
	return &authapi.AuthResponse{
		Result: "ok",
		Token:  "abcd123",
	}, nil
}
