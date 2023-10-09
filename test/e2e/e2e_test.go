package e2e

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"path"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/eventstreamapi"
	"github.com/jannfis/argocd-application-agent/server"
	fakecerts "github.com/jannfis/argocd-application-agent/test/fake/certs"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var certTempl = x509.Certificate{
	SerialNumber:          big.NewInt(1),
	KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	BasicConstraintsValid: true,
	NotBefore:             time.Now().Add(-1 * time.Hour),
	NotAfter:              time.Now().Add(1 * time.Hour),
}

var testNamespace = "default"

func newStreamingClient(t *testing.T, s *server.Server) (eventstreamapi.EventStreamClient, *grpc.ClientConn) {
	t.Helper()
	tlsC := &tls.Config{InsecureSkipVerify: true}
	creds := credentials.NewTLS(tlsC)
	conn, err := grpc.Dial(s.Listener().Address(),
		grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	return eventstreamapi.NewEventStreamClient(conn), conn
}

func Test_EndToEnd(t *testing.T) {
	tempDir := t.TempDir()
	templ := certTempl
	fakecerts.WriteFakeRSAKeyPair(t, path.Join(tempDir, "test-cert"), templ)
	appC := fakeappclient.NewSimpleClientset()
	errch := make(chan error)

	s, err := server.NewServer(appC, testNamespace,
		server.WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
		server.WithListenerPort(0),
		server.WithListenerAddress("127.0.0.1"),
		server.WithShutDownGracePeriod(2*time.Second),
	)
	require.NoError(t, err)

	err = s.ServeGRPC(context.Background(), errch)
	assert.NoError(t, err)

	token, err := s.Issuer().Issue("default", 1*time.Minute)
	require.NoError(t, err)

	clientCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	clientCtx = metadata.AppendToOutgoingContext(clientCtx, "authorization", token)

	client, conn := newStreamingClient(t, s)
	defer conn.Close()

	sub, err := client.Subscribe(clientCtx)
	require.NotNil(t, sub)
	require.NoError(t, err)

	waitc := make(chan struct{})
	serverRunning := true
	appsCreated := 0

	go func() {
		numRecvd := 0
		for {
			ev, err := sub.Recv()
			require.NoError(t, err)
			numRecvd += 1
			logrus.WithField("module", "test-client").Infof("Received event %v", ev)
			if numRecvd >= 4 {
				logrus.Infof("Finished receiving")
				break
			}
		}
		close(waitc)
	}()

	for serverRunning {
		select {
		case <-clientCtx.Done():
			logrus.Infof("Done")
			serverRunning = false
		case <-waitc:
			logrus.Infof("Client closed the connection")
			serverRunning = false
		default:
			if appsCreated > 4 {
				log().Infof("Reached limit")
				serverRunning = false
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
	s.ShutDown()
}

func log() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}
