package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/eventstreamapi"
	fakecerts "github.com/jannfis/argocd-application-agent/test/fake/certs"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newStreamingClient(t *testing.T, s *Server) (eventstreamapi.EventStreamClient, *grpc.ClientConn) {
	t.Helper()
	tlsC := &tls.Config{InsecureSkipVerify: true}
	creds := credentials.NewTLS(tlsC)
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", s.listener.host, s.listener.port),
		grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	return eventstreamapi.NewEventStreamClient(conn), conn
}

func Test_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	templ := certTempl
	fakecerts.WriteFakeRSAKeyPair(t, path.Join(tempDir, "test-cert"), templ)
	appC := fakeappclient.NewSimpleClientset()

	s, err := NewServer(appC, testNamespace,
		WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
		WithListenerPort(0),
		WithListenerAddress("127.0.0.1"),
		WithShutDownGracePeriod(2*time.Second),
	)
	require.NoError(t, err)
	errch := make(chan error)
	err = s.ServeGRPC(context.Background(), errch)
	assert.NoError(t, err)
	// ticker := time.NewTicker(500 * time.Millisecond)
	// timeout := time.NewTicker(2 * time.Second)
	// agentName := "testagent"
	// if !s.queues.HasQueuePair(agentName) {
	// 	s.queues.Create(agentName)
	// }

	client, conn := newStreamingClient(t, s)
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sub, err := client.Subscribe(ctx)
	require.NotNil(t, sub)
	require.NoError(t, err)
	waitc := make(chan struct{})
	serverRunning := true
	appsCreated := 0
	for serverRunning {
		select {
		case <-sub.Context().Done():
			logrus.Infof("Done")
			sub.CloseSend()
			s.Stop()
			serverRunning = false
		case <-waitc:
			sub.CloseSend()
			serverRunning = false
		default:
			if appsCreated > 4 {
				continue
			}
			time.Sleep(100 * time.Millisecond)
			_, err := appC.ArgoprojV1alpha1().Applications(testNamespace).Create(context.TODO(), &v1alpha1.Application{
				ObjectMeta: v1.ObjectMeta{
					Name:      fmt.Sprintf("app%d", appsCreated+1),
					Namespace: testNamespace,
				},
			}, v1.CreateOptions{})
			require.NoError(t, err)
			appsCreated += 1
		}
	}

	appC.ArgoprojV1alpha1().Applications("")
	// select {
	// case <-ticker.C:
	// 	r, err := client.Version(context.Background(), &versionapi.VersionRequest{})
	// 	require.NoError(t, err)
	// 	assert.Equal(t, r.Version, version.QualifiedVersion())
	// 	s.Stop()
	// 	ticker.Stop()
	// case <-timeout.C:
	// 	t.Fatalf("Timed out waiting for cancel")
	// case err = <-errch:
	// 	require.NoError(t, err)
	// default:
	// 	time.Sleep(100 * time.Millisecond)
	// }

}
