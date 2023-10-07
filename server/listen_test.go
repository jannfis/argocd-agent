package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/jannfis/argocd-application-agent/internal/version"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/versionapi"
	fakecerts "github.com/jannfis/argocd-application-agent/test/fake/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

// type slowedClient struct{}

// func (c *slowedClient) RoundTrip(r *http.Request) (*http.Response, error) {
// 	t := &http.Transport{
// 		TLSClientConfig: &tls.Config{
// 			InsecureSkipVerify: true,
// 		},
// 	}
// 	return t.RoundTrip(r)
// }

func Test_parseAddress(t *testing.T) {
	tc := []struct {
		address string
		host    string
		port    int
		valid   bool
	}{
		{"127.0.0.1:8080", "127.0.0.1", 8080, true},
		{"[::1]:8080", "[::1]", 8080, true},
		{"127.0.0.1:203201", "", 0, false},
		{"[some]:host]:8080", "", 0, false},
	}

	for _, tt := range tc {
		host, port, err := parseAddress(tt.address)
		if !tt.valid {
			assert.Error(t, err)
			assert.Equal(t, tt.host, host)
			assert.Equal(t, tt.port, port)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.host, host)
			assert.Equal(t, tt.port, port)
		}
	}
}

func Test_Listen(t *testing.T) {
	tempDir := t.TempDir()
	templ := certTempl
	fakecerts.WriteFakeRSAKeyPair(t, path.Join(tempDir, "test-cert"), templ)
	t.Run("Auto-select port for listener", func(t *testing.T) {
		s, err := NewServer(
			WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
			WithListenerPort(0),
			WithListenerAddress("127.0.0.1"),
		)
		require.NoError(t, err)
		err = s.Listen(context.Background(), wait.Backoff{Duration: 100 * time.Millisecond, Steps: 2})
		require.NoError(t, err)
		defer s.listener.l.Close()
		assert.Equal(t, "127.0.0.1", s.listener.host)
		assert.NotZero(t, s.listener.port)
	})

	t.Run("Listen on privileged port", func(t *testing.T) {
		s, err := NewServer(
			WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
			WithListenerPort(443),
			WithListenerAddress("127.0.0.1"),
		)
		require.NoError(t, err)
		err = s.Listen(context.Background(), wait.Backoff{Duration: 100 * time.Millisecond, Steps: 2})
		require.Error(t, err)
		assert.Nil(t, s.listener)
	})

}

func Test_Serve(t *testing.T) {
	tempDir := t.TempDir()
	templ := certTempl
	fakecerts.WriteFakeRSAKeyPair(t, path.Join(tempDir, "test-cert"), templ)
	s, err := NewServer(
		WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
		WithListenerPort(0),
		WithListenerAddress("127.0.0.1"),
		WithGracePeriod(2*time.Second),
	)
	require.NoError(t, err)
	errch := make(chan error)
	err = s.ServeGRPC(context.Background(), errch)
	assert.NoError(t, err)
	ticker := time.NewTicker(500 * time.Millisecond)
	timeout := time.NewTicker(2 * time.Second)

	for s.server != nil || s.grpcServer != nil {
		select {
		case <-ticker.C:
			tlsC := &tls.Config{InsecureSkipVerify: true}
			creds := credentials.NewTLS(tlsC)
			conn, err := grpc.Dial(fmt.Sprintf("%s:%d", s.listener.host, s.listener.port),
				grpc.WithTransportCredentials(creds))
			require.NoError(t, err)
			defer conn.Close()
			client := versionapi.NewVersionClient(conn)
			r, err := client.Version(context.Background(), &versionapi.VersionRequest{})
			require.NoError(t, err)
			assert.Equal(t, r.Version, version.QualifiedVersion())
			s.Stop()
			ticker.Stop()
		case <-timeout.C:
			t.Fatalf("Timed out waiting for cancel")
		case err = <-errch:
			require.NoError(t, err)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
