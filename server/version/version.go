package version

import (
	"context"

	"github.com/jannfis/argocd-application-agent/internal/version"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/versionapi"
)

type server struct {
	versionapi.UnimplementedVersionServer
}

func NewServer() *server {
	return &server{}
}

func (s *server) Version(ctx context.Context, r *versionapi.VersionRequest) (*versionapi.VersionResponse, error) {
	return &versionapi.VersionResponse{Version: version.QualifiedVersion()}, nil
}
