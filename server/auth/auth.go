package auth

import (
	"context"

	"github.com/jannfis/argocd-application-agent/internal/auth"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/authapi"
	"github.com/jannfis/argocd-application-agent/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	authapi.UnimplementedAuthenticationServer
	authMethods *auth.Methods
	options     *ServerOptions
}

type ServerOptions struct {
}

type ServerOption func(o *ServerOptions) error

// NewServer creates a new instance of an authentication server with the given
// authentication methods and options.
func NewServer(authMethods *auth.Methods, opts ...ServerOption) *server {
	s := &server{}
	s.options = &ServerOptions{}
	if authMethods != nil {
		s.authMethods = authMethods
	} else {
		s.authMethods = auth.NewMethods()
	}
	for _, o := range opts {
		o(s.options)
	}
	return s
}

func authenticationFailed(reason string) *authapi.AuthResponse {
	return &authapi.AuthResponse{
		Result: types.AuthResultUnauthorized,
	}
}

func (s *server) Authenticate(ctx context.Context, a *authapi.AuthRequest) (*authapi.AuthResponse, error) {
	am := s.authMethods.AuthMethod(a.Method)
	if am == nil {
		return authenticationFailed(""), status.Error(codes.Unauthenticated, "unsupported authentication method")
	}
	ok, err := am.Authenticate(a.Credentials)
	if !ok || err != nil {
		return authenticationFailed(""), status.Error(codes.Unauthenticated, "authentication failed")
	}
	return &authapi.AuthResponse{
		Result: types.AuthResultOK,
		Token:  "abcd123",
	}, nil
}
