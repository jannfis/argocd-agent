package version

import (
	"context"
	"testing"

	"github.com/jannfis/argocd-application-agent/internal/version"
	versionapi "github.com/jannfis/argocd-application-agent/pkg/api/grpc/version"
	"github.com/stretchr/testify/assert"
)

func Test_Version(t *testing.T) {
	t.Run("Get version identifier", func(t *testing.T) {
		s := NewServer()
		r, err := s.Version(context.Background(), &versionapi.VersionRequest{})
		assert.NoError(t, err)
		assert.Equal(t, version.QualifiedVersion(), r.Version)
	})
}
