package server

import (
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"time"
)

// supportedTLSVersion is a list of TLS versions we support
var supportedTLSVersion map[string]int = map[string]int{
	"tls1.1": tls.VersionTLS11,
	"tls1.2": tls.VersionTLS12,
	"tls1.3": tls.VersionTLS13,
}

type ServerOptions struct {
	serverName    string
	port          int
	address       string
	tlsCert       string
	tlsKey        string
	tlsCiphers    *tls.CipherSuite
	tlsMinVersion int
	gracePeriod   time.Duration
	namespaces    []string
	signingKey    *rsa.PrivateKey
	unauthMethods map[string]bool
	serveGRPC     bool
	serveREST     bool
}

type ServerOption func(o *ServerOptions) error

// defaultOptions returns a set of default options for the server
func defaultOptions() *ServerOptions {
	return &ServerOptions{
		port:          443,
		address:       "",
		tlsMinVersion: tls.VersionTLS13,
		unauthMethods: make(map[string]bool),
	}
}

// WithTokenSigningKey sets the RSA private key to use for signing the tokens
// issued by the Server
func WithTokenSigningKey(key *rsa.PrivateKey) ServerOption {
	return func(o *ServerOptions) error {
		o.signingKey = key
		return nil
	}
}

// WithListenerPort sets the listening port for the server. If the port is not
// valid, an error is returned.
func WithListenerPort(port int) ServerOption {
	return func(o *ServerOptions) error {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port must be between 0 and 65535")
		}
		o.port = port
		return nil
	}
}

// WithListenerAddress sets the address the server should listen on.
func WithListenerAddress(host string) ServerOption {
	return func(o *ServerOptions) error {
		o.address = host
		return nil
	}
}

// WithTLSKeyPair configures the TLS certificate and private key to be used by
// the server. The function will not check whether the files exists, or if they
// contain valid data because it is assumed that they may be created at a later
// point in time.
func WithTLSKeyPair(certPath, keyPath string) ServerOption {
	return func(o *ServerOptions) error {
		o.tlsCert = certPath
		o.tlsKey = keyPath
		return nil
	}
}

// WithTLSCipherSuite configures the TLS cipher suite to be used by the server.
// If an unknown cipher suite is specified, an error is returned.
func WithTLSCipherSuite(cipherSuite string) ServerOption {
	return func(o *ServerOptions) error {
		for _, cs := range tls.CipherSuites() {
			if cs.Name == cipherSuite {
				o.tlsCiphers = cs
				return nil
			}
		}
		return fmt.Errorf("no such cipher suite: %s", cipherSuite)
	}
}

// WithMinimumTLSVersion configures the minimum TLS version to be accepted by
// the server.
func WithMinimumTLSVersion(version string) ServerOption {
	return func(o *ServerOptions) error {
		v, ok := supportedTLSVersion[version]
		if !ok {
			return fmt.Errorf("TLS version %s is not supported", version)
		}
		o.tlsMinVersion = v
		return nil
	}
}

// WithShutDownGracePeriod configures how long the server should wait for
// client connections to close during shutdown. If d is 0, the server will
// not use a grace period for shutdown but instead close immediately.
func WithShutDownGracePeriod(d time.Duration) ServerOption {
	return func(o *ServerOptions) error {
		o.gracePeriod = d
		return nil
	}
}

// WithNamespaces sets an
func WithNamespaces(namespaces ...string) ServerOption {
	return func(o *ServerOptions) error {
		o.namespaces = namespaces
		return nil
	}
}

func WithGRPC(serveGRPC bool) ServerOption {
	return func(o *ServerOptions) error {
		o.serveGRPC = serveGRPC
		return nil
	}
}

func WithREST(serveREST bool) ServerOption {
	return func(o *ServerOptions) error {
		o.serveREST = serveREST
		return nil
	}
}

func WithServerName(serverName string) ServerOption {
	return func(o *ServerOptions) error {
		o.serverName = serverName
		return nil
	}
}
