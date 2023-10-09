package token

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer issues and validates JSON web tokens (JWT) for authentication
type Issuer struct {
	name       string
	privateKey *rsa.PrivateKey
}

// IssuerOption is a function to set options for the Issuer
type IssuerOption func(i *Issuer) error

// WithPrivateRSAKey sets the private RSA for the Issuer
func WithPrivateRSAKey(key *rsa.PrivateKey) IssuerOption {
	return func(i *Issuer) error {
		i.privateKey = key
		return nil
	}
}

// WithPrivateRSAKeyFromFile loads a PEM-encoded RSA private key from path and
// sets it as the private RSA key for the Issuer
func WithPrivateRSAKeyFromFile(path string) IssuerOption {
	return func(i *Issuer) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read RSA private key: %w", err)
		}
		p, _ := pem.Decode(data)
		if p == nil {
			return fmt.Errorf("no valid PEM data found in %s", path)
		}
		key, err := x509.ParsePKCS1PrivateKey(p.Bytes)
		if err != nil {
			return fmt.Errorf("could not parse RSA private key data from %s: %w", path, err)
		}
		i.privateKey = key
		return nil
	}
}

// NewIssuer creates a new instance of Issuer, which is used to issue JWTs
// to authenticated clients and to validate incoming JWTs.
func NewIssuer(name string, opts ...IssuerOption) (*Issuer, error) {
	iss := &Issuer{
		name: name,
	}
	for _, o := range opts {
		if err := o(iss); err != nil {
			return nil, err
		}
	}
	return iss, nil
}

// Issue creates and signs a new token for client, which is valid through the
// duration specified as exp. The result is returned as a string.
func (i *Issuer) Issue(client string, exp time.Duration) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		Issuer:    i.name,
		Subject:   client,
		Audience:  jwt.ClaimStrings{i.name},
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
	})
	return t.SignedString(i.privateKey)
}

func (i *Issuer) Validate(token string) (jwt.Claims, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		var pubKey *rsa.PublicKey = &i.privateKey.PublicKey
		if t.Method != jwt.SigningMethodRS512 {
			return nil, fmt.Errorf("token isn't signed with %s method", jwt.SigningMethodRS512)
		}
		return pubKey, nil
	}
	t, err := jwt.Parse(token, keyFunc,
		jwt.WithAudience(i.name),
		jwt.WithIssuer(i.name),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS512.Name}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not validate token: %w", err)
	}
	return t.Claims, nil
}
