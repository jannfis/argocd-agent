package server

import (
	context "context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Server struct {
	options    *ServerOptions
	tlsConfig  *tls.Config
	listener   *Listener
	server     *http.Server
	grpcServer *grpc.Server
}

func NewServer(opts ...ServerOption) (*Server, error) {
	options := defaultOptions()
	for _, o := range opts {
		err := o(options)
		if err != nil {
			return nil, err
		}
	}
	s := &Server{options: options}
	return s, nil
}

func (s *Server) Run() error {
	tlsConfig, err := s.loadTLSConfig()
	if err != nil {
		return err
	}
	s.tlsConfig = tlsConfig
	return nil
}

func (s *Server) Stop() error {
	var err error
	if s.server != nil {
		if s.options.gracePeriod > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), s.options.gracePeriod)
			defer cancel()
			log.Infof("Server shutdown requested, allowing client connections to shut down for %v", s.options.gracePeriod)
			err = s.server.Shutdown(ctx)
		} else {
			log.Infof("Closing server")
			err = s.server.Close()
		}
		s.server = nil
	} else if s.grpcServer != nil {
		s.grpcServer.Stop()
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
			log.Warnf("Server certificate has expired on %s", cert.NotAfter.Format(time.RFC1123Z))
		}
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return tlsConfig, nil
}
