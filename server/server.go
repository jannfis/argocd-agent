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
	"github.com/jannfis/argocd-agent/internal/appinformer"
	"github.com/jannfis/argocd-agent/internal/application"
	"github.com/jannfis/argocd-agent/internal/auth"
	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/jannfis/argocd-agent/internal/queue"
	"github.com/jannfis/argocd-agent/internal/token"
	"github.com/jannfis/argocd-agent/server/backend/kubernetes"
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
	namespace   string
	issuer      *token.Issuer
	noauth      map[string]bool // noauth contains endpoints accessible without authentication
	ctx         context.Context
	ctxCancel   context.CancelFunc
	appManager  *application.Manager
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

	issuer, err := token.NewIssuer("argocd-agent-server", token.WithRSAPrivateKey(options.signingKey))
	if err != nil {
		return nil, err
	}

	s := &Server{
		options:     options,
		authMethods: auth.NewMethods(),
		queues:      queue.NewSendRecvQueues(),
		namespace:   namespace,
		issuer:      issuer,
		noauth: map[string]bool{
			"/versionapi.Version/Version":          true,
			"/authapi.Authentication/Authenticate": true,
		},
	}

	informerOpts := []appinformer.AppInformerOption{
		appinformer.WithNamespaces(options.namespaces...),
		appinformer.WithNewAppCallback(s.newAppCallback),
	}

	managerOpts := []application.ManagerOption{
		application.WithAllowUpsert(true),
	}

	if s.options.metricsPort > 0 {
		informerOpts = append(informerOpts, appinformer.WithMetrics(metrics.NewApplicationWatcherMetrics()))
		managerOpts = append(managerOpts, application.WithMetrics(metrics.NewApplicationClientMetrics()))
	}

	informer := appinformer.NewAppInformer(appClient,
		s.namespace,
		informerOpts...,
	)

	s.appManager = application.NewManager(kubernetes.NewKubernetesBackend(appClient, informer),
		managerOpts...,
	)

	return s, nil
}

// Start starts the Server s and its listeners in their own go routines. Any
// error during startup, before the go routines are running, will be returned
// immediately. Errors during the runtime will be propagated via errch.
func (s *Server) Start(ctx context.Context, errch chan error) error {
	s.ctx, s.ctxCancel = context.WithCancel(ctx)
	if s.options.serveGRPC {
		if err := s.serveGRPC(s.ctx, errch); err != nil {
			return err
		}
	}

	if s.options.metricsPort > 0 {
		metrics.StartMetricsServer(metrics.WithListener("", s.options.metricsPort))
	}

	err := s.StartEventProcessor(s.ctx)
	if err != nil {
		return nil
	}

	return nil
}

// Shutdown shuts down the server s. If no server is running, or shutting down
// results in an error, an error is returned.
func (s *Server) Shutdown() error {
	var err error
	// Cancel server-wide context
	s.ctxCancel()

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

// Listener returns the listener of Server s
func (s *Server) Listener() *Listener {
	return s.listener
}

// TokenIssuer returns the token issuer of Server s
func (s *Server) TokenIssuer() *token.Issuer {
	return s.issuer
}

func log() *logrus.Entry {
	return logrus.WithField("module", "server")
}

func (s *Server) AuthMethods() *auth.Methods {
	return s.authMethods
}

func (s *Server) Queues() *queue.SendRecvQueues {
	return s.queues
}

func (s *Server) AppManager() *application.Manager {
	return s.appManager
}
