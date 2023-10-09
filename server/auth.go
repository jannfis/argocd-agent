package server

import (
	"context"

	"github.com/jannfis/argocd-application-agent/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Authenticate is used as a gRPC interceptor to decide whether a request is
// authenticated or not. If the request is authenticated, Authenticate will
// also augment the Context of the request with additional information about
// the client, that can later be evaluated by the server's RPC methods and
// streams.
//
// If the request turns out to be unauthenticated, Authenticate will
// return an appropriate error.
func (s *Server) Authenticate(ctx context.Context) (context.Context, error) {
	logCtx := log().WithField("module", "AuthHandler")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "could not get metadata from request")
	}
	jwt, ok := md["authorization"]
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no authentication data found")
	}
	claims, err := s.issuer.Validate(jwt[0])
	if err != nil {
		logCtx.Warnf("Error validating token: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid authentication data")
	}

	agentName, err := claims.GetSubject()
	if err != nil {
		logCtx.Warnf("Could not get subject from token: %v", err)
		return nil, status.Error(codes.Unauthenticated, "invalid authentication data")
	}

	// claims at this point is validated and we can propagate values to the
	// context.
	authCtx := context.WithValue(ctx, types.ContextAgentIdentifier, agentName)
	if !s.queues.HasQueuePair(agentName) {
		logCtx.Tracef("Creating a new queue pair for client %s", agentName)
		s.queues.Create(agentName)
	}
	logCtx.WithField("client", agentName).Tracef("Client passed authentication")
	return authCtx, nil
}
