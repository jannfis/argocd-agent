package server

import (
	context "context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/jannfis/argocd-application-agent/internal/appinformer"
	"github.com/jannfis/argocd-application-agent/internal/auth"
	"github.com/jannfis/argocd-application-agent/internal/queue"
	"github.com/jannfis/argocd-application-agent/internal/token"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server struct {
	options     *ServerOptions
	tlsConfig   *tls.Config
	listener    *Listener
	server      *http.Server
	grpcServer  *grpc.Server
	authMethods *auth.Methods
	queues      *queue.SendRecvQueues
	informer    *appinformer.AppInformer
	informerCh  chan struct{}
	namespace   string
	issuer      *token.Issuer
}

func NewServer(appClient appclientset.Interface, namespace string, opts ...ServerOption) (*Server, error) {
	options := defaultOptions()
	for _, o := range opts {
		err := o(options)
		if err != nil {
			return nil, err
		}
	}

	// The server supports generating and using a volatile signing keys for the
	// tokens it issues. This should not be used in production.
	if options.signingKey == nil {
		log().Warnf("Generating and using a volatile token signing key - multiple replicas not possible")
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("could not generate signing key: %v", err)
		}
		options.signingKey = key
	}

	issuer, err := token.NewIssuer("argocd-agent-server", token.WithPrivateRSAKey(options.signingKey))
	if err != nil {
		return nil, err
	}

	s := &Server{
		options:     options,
		authMethods: auth.NewMethods(),
		queues:      queue.NewSendRecvQueues(),
		namespace:   namespace,
		informerCh:  make(chan struct{}),
		issuer:      issuer,
	}

	s.informer = appinformer.NewAppInformer(appClient,
		s.namespace,
		appinformer.WithNamespaces(options.namespaces...),
		appinformer.WithNewAppCallback(s.newAppCallback),
	)

	return s, nil
}

// ShutDown shuts down the server s. If no server is running, or shutting down
// results in an error, an error is returned.
func (s *Server) ShutDown() error {
	var err error
	if s.server != nil {
		if s.options.gracePeriod > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), s.options.gracePeriod)
			defer cancel()
			log().Infof("Server shutdown requested, allowing client connections to shut down for %v", s.options.gracePeriod)
			err = s.server.Shutdown(ctx)
		} else {
			log().Infof("Closing server")
			err = s.server.Close()
		}
		s.server = nil
	} else if s.grpcServer != nil {
		log().Infof("Shutting down server")
		s.grpcServer.Stop()
		s.grpcServer = nil
	} else {
		return fmt.Errorf("no server running")
	}
	return err
}

func (s *Server) loadTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(s.options.tlsCert, s.options.tlsKey)
	if err != nil {
		return nil, fmt.Errorf("could not load X509 keypair: %w", err)
	}
	for _, c := range cert.Certificate {
		cert, err := x509.ParseCertificate(c)
		if err != nil {
			return nil, fmt.Errorf("could not parse certificate from %s: %w", s.options.tlsCert, err)
		}
		if !cert.NotAfter.After(time.Now()) {
			log().Warnf("Server certificate has expired on %s", cert.NotAfter.Format(time.RFC1123Z))
		}
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return tlsConfig, nil
}

func (s *Server) Listener() *Listener {
	return s.listener
}

func (s *Server) Issuer() *token.Issuer {
	return s.issuer
}

func log() *logrus.Entry {
	return logrus.WithField("module", "server")
}
