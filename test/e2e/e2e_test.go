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
	"github.com/jannfis/argocd-agent/internal/auth/userpass"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/authapi"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/eventstreamapi"
	"github.com/jannfis/argocd-agent/server"
	"github.com/jannfis/argocd-agent/server/backend"
	fakecerts "github.com/jannfis/argocd-agent/test/fake/certs"
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

func newConn(t *testing.T, appC *fakeappclient.Clientset) (*grpc.ClientConn, *server.Server) {
	t.Helper()
	tempDir := t.TempDir()
	templ := certTempl
	fakecerts.WriteSelfSignedCert(t, path.Join(tempDir, "test-cert"), templ)
	errch := make(chan error)

	s, err := server.NewServer(appC, testNamespace,
		server.WithTLSKeyPair(path.Join(tempDir, "test-cert.crt"), path.Join(tempDir, "test-cert.key")),
		server.WithListenerPort(0),
		server.WithListenerAddress("127.0.0.1"),
		server.WithShutDownGracePeriod(2*time.Second),
		server.WithGRPC(true),
		server.WithEventProcessors(10),
	)
	require.NoError(t, err)
	err = s.Start(context.Background(), errch)
	assert.NoError(t, err)

	am := userpass.NewUserPassAuthentication()
	am.UpsertUser("default", "password")
	s.AuthMethods().RegisterMethod("userpass", am)

	tlsC := &tls.Config{InsecureSkipVerify: true}
	creds := credentials.NewTLS(tlsC)
	conn, err := grpc.Dial(s.Listener().Address(),
		grpc.WithTransportCredentials(creds))
	require.NoError(t, err)
	return conn, s
}

// func newStreamingClient(t *testing.T, conn) (eventstreamapi.EventStreamClient, *grpc.ClientConn) {
// 	t.Helper()
// 	return eventstreamapi.NewEventStreamClient(conn), conn
// }

func Test_EndToEnd_Subscribe(t *testing.T) {
	// token, err := s.TokenIssuer().Issue("default", 1*time.Minute)
	// require.NoError(t, err)

	appC := fakeappclient.NewSimpleClientset()
	conn, s := newConn(t, appC)
	defer conn.Close()

	authC := authapi.NewAuthenticationClient(conn)
	eventC := eventstreamapi.NewEventStreamClient(conn)

	clientCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get authentication token and store in context
	authr, err := authC.Authenticate(clientCtx, &authapi.AuthRequest{Method: "userpass", Credentials: map[string]string{
		userpass.ClientIDField:     "default",
		userpass.ClientSecretField: "password",
	}})
	require.NoError(t, err)
	clientCtx = metadata.AppendToOutgoingContext(clientCtx, "authorization", authr.Token)

	sub, err := eventC.Subscribe(clientCtx)
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
	s.Shutdown()
}

func Test_EndToEnd_Push(t *testing.T) {
	appC := fakeappclient.NewSimpleClientset()
	conn, s := newConn(t, appC)
	defer conn.Close()
	authC := authapi.NewAuthenticationClient(conn)
	eventC := eventstreamapi.NewEventStreamClient(conn)

	clientCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Get authentication token and store in context
	authr, err := authC.Authenticate(clientCtx, &authapi.AuthRequest{Method: "userpass", Credentials: map[string]string{
		userpass.ClientIDField:     "default",
		userpass.ClientSecretField: "password",
	}})
	require.NoError(t, err)
	clientCtx = metadata.AppendToOutgoingContext(clientCtx, "authorization", authr.Token)

	pushc, err := eventC.Push(clientCtx)
	require.NoError(t, err)
	start := time.Now()
	for i := 0; i < 10; i += 1 {
		pushc.Send(&eventstreamapi.Event{
			Event: eventstreamapi.EventType_Event_UpdateApp,
			Application: &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{
				Name:      fmt.Sprintf("test%d", i),
				Namespace: "default",
			}},
		})
	}
	summary, err := pushc.CloseAndRecv()
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int32(10), summary.Received)
	end := time.Now()

	// Wait until the context is done
	<-clientCtx.Done()

	log().Infof("Took %v to process", end.Sub(start))
	s.Shutdown()

	// Should have been grabbed by queue processor
	q := s.Queues()
	assert.Equal(t, 0, q.RecvQ("default").Len())

	// All applications should have been created by now on the server
	apps, err := s.AppManager().Backend.List(context.Background(), backend.ApplicationSelector{})
	assert.NoError(t, err)
	assert.Len(t, apps, 10)
}

func log() *logrus.Entry {
	return logrus.WithField("TEST", "test")
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
