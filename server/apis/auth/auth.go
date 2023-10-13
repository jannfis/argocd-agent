package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/jannfis/argocd-agent/internal/auth"
	"github.com/jannfis/argocd-agent/internal/clock"
	"github.com/jannfis/argocd-agent/internal/issuer"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/authapi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	authapi.UnimplementedAuthenticationServer
	authMethods *auth.Methods
	issuer      issuer.Issuer
	options     *ServerOptions
	clock       clock.Clock
}

const (
	accessTokenValidity  = 5 * time.Minute
	refreshTokenValidity = 24 * time.Hour
)

const (
	authFailedMessage = "authentication failed"
)

type ServerOptions struct {
}

type ServerOption func(o *ServerOptions) error

// NewServer creates a new instance of an authentication server with the given
// authentication methods and options.
func NewServer(authMethods *auth.Methods, iss issuer.Issuer, opts ...ServerOption) *server {
	s := &server{}
	s.options = &ServerOptions{}
	if authMethods != nil {
		s.authMethods = authMethods
	} else {
		s.authMethods = auth.NewMethods()
	}
	if iss == nil {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil
		}
		iss, err = issuer.NewIssuer("default", issuer.WithRSAPrivateKey(key))
		if err != nil {
			return nil
		}
	}
	s.issuer = iss
	for _, o := range opts {
		o(s.options)
	}
	s.clock = clock.StandardClock()
	return s
}

func (s *server) issueTokens(subject string, refresh bool) (accessToken string, refreshToken string, err error) {
	accessToken, err = s.issuer.IssueAccessToken(subject, accessTokenValidity)
	if err != nil {
		return "", "", status.Error(codes.Internal, "unable to generate a token")
	}
	if refresh {
		refreshToken, err = s.issuer.IssueRefreshToken(subject, refreshTokenValidity)
		if err != nil {
			return "", "", status.Error(codes.Internal, "unable to generate a token")
		}
	}
	return accessToken, refreshToken, nil
}

// Authenticate provides an authz endpoint for the Server. The client is
// supposed to specify the authentication method and the credentials to use.
//
// A Server may support one or more authentication methods, and if the authz
// request succeeds, a JWT will be issued to the client.
func (s *server) Authenticate(ctx context.Context, ar *authapi.AuthRequest) (*authapi.AuthResponse, error) {
	logCtx := log().WithField("method", "Authenticate").WithField("authmethod", ar.Method)
	am := s.authMethods.Method(ar.Method)
	if am == nil {
		logCtx.Info("unknown authentication method")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}
	clientID, err := am.Authenticate(ar.Credentials)
	if clientID == "" || err != nil {
		logCtx.WithError(err).WithField("client", clientID).Info("client authentication failed")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}
	accessToken, refreshToken, err := s.issueTokens(clientID, true)
	if err != nil {
		logCtx.WithError(err).Warnf("Unable to generate token")
		return nil, err
	}
	return &authapi.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken issues a new access token when the client presents a valid
// refresh token. If the refresh token is only valid for 10 minutes or less,
// a new refresh token will be issued as well.
func (s *server) RefreshToken(ctx context.Context, r *authapi.RefreshTokenRequest) (*authapi.AuthResponse, error) {
	logCtx := log().WithField("method", "RefreshToken")
	if r.RefreshToken == "" {
		logCtx.Warn("No refresh token supplied")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}

	c, err := s.issuer.ValidateRefreshToken(r.RefreshToken)
	if err != nil {
		logCtx.WithError(err).Warnf("Could not validate refresh token")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}

	// We need the subject of the refresh token to issue a new one
	subj, err := c.GetSubject()
	if err != nil {
		logCtx.WithError(err).Warnf("Could not get subject from refresh token")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}

	// We only want to issue a new refresh token when the old one is close to
	// expiry.
	exp, err := c.GetExpirationTime()
	if err != nil {
		logCtx.WithError(err).Warnf("Could not get exp from refresh token")
		return nil, status.Error(codes.Unauthenticated, authFailedMessage)
	}
	refresh := false
	if s.clock.Until(exp.Time) < 10*time.Minute {
		refresh = true
	}

	accessToken, refreshToken, err := s.issueTokens(subj, refresh)
	if err != nil {
		logCtx.WithError(err).WithField("refresh", refresh).Warnf("Could not issue a new token")
		return nil, status.Error(codes.Internal, "could not issue tokens")
	}
	return &authapi.AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func log() *logrus.Entry {
	return logrus.WithField("module", "grpc.AuthenticationServer")
}
