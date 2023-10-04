package server

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WithPort(t *testing.T) {
	ports := []struct {
		port  int
		valid bool
	}{
		{1, true},
		{0, true},
		{-1, false},
		{65535, true},
		{65536, false},
	}

	for _, tt := range ports {
		opts := &ServerOptions{}
		err := WithListenerPort(tt.port)(opts)
		if tt.valid {
			assert.NoErrorf(t, err, "port %d should be valid", tt.port)
			assert.Equal(t, tt.port, opts.port)
		} else {
			assert.Errorf(t, err, "port %d should be invalid", tt.port)
			assert.Equal(t, 0, opts.port)
		}
	}
}

func Test_WithListenerAddress(t *testing.T) {
	opts := &ServerOptions{}
	err := WithListenerAddress("127.0.0.1")(opts)
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", opts.address)
}

func Test_WithTLSCipherSuite(t *testing.T) {
	t.Run("All valid cipher suites", func(t *testing.T) {
		for _, cs := range tls.CipherSuites() {
			opts := &ServerOptions{}
			err := WithTLSCipherSuite(cs.Name)(opts)
			assert.NoError(t, err)
			assert.Equal(t, cs, opts.tlsCiphers)
		}
	})

	t.Run("Invalid cipher suite", func(t *testing.T) {
		opts := &ServerOptions{}
		err := WithTLSCipherSuite("cowabunga")(opts)
		assert.Error(t, err)
		assert.Nil(t, opts.tlsCiphers)
	})
}

func Test_WithMinimumTLSVersion(t *testing.T) {
	t.Run("All valid minimum cipher suites", func(t *testing.T) {
		for k, v := range supportedTLSVersion {
			opts := &ServerOptions{}
			err := WithMinimumTLSVersion(k)(opts)
			assert.NoError(t, err)
			assert.Equal(t, v, opts.tlsMinVersion)
		}
	})

	t.Run("Invalid minimum cipher suites", func(t *testing.T) {
		for _, v := range []string{"tls1.0", "ssl3.0", "invalid", "tls"} {
			opts := &ServerOptions{}
			err := WithMinimumTLSVersion(v)(opts)
			assert.Error(t, err)
			assert.Equal(t, 0, opts.tlsMinVersion)
		}
	})
}
