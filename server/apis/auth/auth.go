package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/jannfis/argocd-agent/internal/auth"
	"github.com/jannfis/argocd-agent/internal/token"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/authapi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	authapi.UnimplementedAuthenticationServer
	authMethods *auth.Methods
	issuer      *token.Issuer
	options     *ServerOptions
}

type ServerOptions struct {
}

type ServerOption func(o *ServerOptions) error

// NewServer creates a new instance of an authentication server with the given
// authentication methods and options.
func NewServer(authMethods *auth.Methods, issuer *token.Issuer, opts ...ServerOption) *server {
	s := &server{}
	s.options = &ServerOptions{}
	if authMethods != nil {
		s.authMethods = authMethods
	} else {
		s.authMethods = auth.NewMethods()
	}
	if issuer == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil
		}
		issuer, err = token.NewIssuer("default", token.WithRSAPrivateKey(key))
		if err != nil {
			return nil
		}
	}
	s.issuer = issuer
	for _, o := range opts {
		o(s.options)
	}
	return s
}

// Authenticate provides an authz endpoint for the Server. The client is
// supposed to specify the authentication method and the credentials to use.
//
// A Server may support one or more authentication methods, and if the authz
// request succeeds, a JWT will be issued to the client.
func (s *server) Authenticate(ctx context.Context, ar *authapi.AuthRequest) (*authapi.AuthResponse, error) {
	am := s.authMethods.Method(ar.Method)
	if am == nil {
		return nil, status.Error(codes.Unauthenticated, "unsupported authentication method")
	}
	clientID, err := am.Authenticate(ar.Credentials)
	if clientID == "" || err != nil {
		return nil, status.Error(codes.Unauthenticated, "authentication failed")
	}
	accessToken, err := s.issuer.IssueAccessToken(clientID, 1*time.Hour)
	if err != nil {
		log().WithField("method", "Authenticate").WithError(err).Warnf("Unable to generate access token")
		return nil, status.Error(codes.Internal, "unable to generate a token")
	}
	refreshToken, err := s.issuer.IssueRefreshToken(clientID, 8*time.Hour)
	if err != nil {
		log().WithField("method", "Authenticate").WithError(err).Warnf("Unable to generate refresh token")
		return nil, status.Error(codes.Internal, "unable to generate a token")
	}
	return &authapi.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *server) RefreshToken(ctx context.Context, r *authapi.RefreshTokenRequest) (*authapi.AuthResponse, error) {
	return nil, nil
}

func log() *logrus.Entry {
	return logrus.WithField("module", "grpc.AuthenticationServer")
}
