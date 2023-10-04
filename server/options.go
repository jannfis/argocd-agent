package server

import (
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
	port          int
	address       string
	tlsCert       string
	tlsKey        string
	tlsCiphers    *tls.CipherSuite
	tlsMinVersion int
	gracePeriod   time.Duration
}

type ServerOption func(o *ServerOptions) error

// defaultOptions returns a set of default options for the server
func defaultOptions() *ServerOptions {
	return &ServerOptions{
		port:          443,
		address:       "",
		tlsCert:       "/etc/tls/certs/server.crt",
		tlsKey:        "/etc/tls/private/server.key",
		tlsMinVersion: tls.VersionTLS13,
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

// WithGracePeriod configures how long the server should wait for connections
// to close during shutdown. If d is 0, the server will not use a grace period
// for shutdown but instead close immediately.
func WithGracePeriod(d time.Duration) ServerOption {
	return func(o *ServerOptions) error {
		o.gracePeriod = d
		return nil
	}
}
