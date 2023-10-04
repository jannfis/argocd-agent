package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

// CreateFakeRSAKeyPair is a test helper that creates a self-signed certificate
// from a template and returns the PEM encoded certificate and key.
func CreateFakeRSAKeyPair(t *testing.T, template x509.Certificate) (certBytes []byte, keyBytes []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error generating key: %v", err)
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("error generating cert: %v", err)
	}
	var keyPem, certPem bytes.Buffer
	err = pem.Encode(&keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err != nil {
		t.Fatalf("error encoding key: %v", err)
	}
	err = pem.Encode(&certPem, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err != nil {
		t.Fatalf("error encoding certificate: %v", err)
	}
	return certPem.Bytes(), keyPem.Bytes()
}

// WriteFakeRSAKeyPair is a test helper to write a self-signed certificate, as
// defined by templ, and its corresponding key in PEM formats to files suffixed
// by ".crt" and ".key" respectively to the path specified by basePath.
func WriteFakeRSAKeyPair(t *testing.T, basePath string, templ x509.Certificate) {
	t.Helper()
	certBytes, keyBytes := CreateFakeRSAKeyPair(t, templ)
	certFile := basePath + ".crt"
	keyFile := basePath + ".key"
	err := os.WriteFile(certFile, certBytes, 0644)
	if err != nil {
		t.Fatalf("error writing certfile: %v", err)
	}
	err = os.WriteFile(keyFile, keyBytes, 0600)
	if err != nil {
		t.Fatalf("error writing keyfile: %v", err)
	}
}
