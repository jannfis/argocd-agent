package server

import (
	context "context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/jannfis/argocd-application-agent/internal/appinformer"
	"github.com/jannfis/argocd-application-agent/internal/auth"
	"github.com/jannfis/argocd-application-agent/internal/queue"
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
	namespace   string
}

func NewServer(appClient appclientset.Interface, namespace string, opts ...ServerOption) (*Server, error) {
	options := defaultOptions()
	for _, o := range opts {
		err := o(options)
		if err != nil {
			return nil, err
		}
	}
	s := &Server{
		options:     options,
		authMethods: auth.NewMethods(),
		queues:      queue.NewSendRecvQueues(),
		namespace:   namespace,
	}

	s.informer = appinformer.NewAppInformer(appClient,
		s.namespace,
		appinformer.WithNamespaces(options.namespaces...),
		appinformer.WithNewAppCallback(s.newAppCallback),
	)

	return s, nil
}

func (s *Server) newAppCallback(app *v1alpha1.Application) {
	logCtx := log().WithField("component", "NewAppCallback")
	if !s.queues.HasQueuePair(app.Namespace) {
		logCtx.Debugf("Creating new queue pair for namespace %s", app.Namespace)
		err := s.queues.Create(app.Namespace)
		if err != nil {
			logCtx.Errorf("could not create new queue pair for namespace %s: %v", app.Namespace, err)
			return
		}
	}
	q := s.queues.SendQ(app.Namespace)
	if q == nil {
		logCtx.Errorf("Help! queue pair for namespace %s disappeared!", app.Namespace)
		return
	}

	q.Add(app)
	logCtx.Tracef("Added app %s to send queue, total length now %d", app.QualifiedName(), q.Len())
}

func (s *Server) Stop() error {
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

func log() *logrus.Entry {
	return logrus.WithField("module", "server")
}
