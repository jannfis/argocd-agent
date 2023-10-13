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

// Issuer issues and validates access and refresh tokens in JWT format. If the
// Issuer is configured with a private key, it can be used to both issue and
// validate tokens. If the Issuer is configured with a public key but not a
// private key, it can be only used to verify tokens. An Issuer should not be
// configured with both, a private and a public key. For Issuers with a private
// key, the public key for validation will be derived from the private key.
type Issuer struct {
	name       string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	atAudience string
	rtAudience string
}

// IssuerOption is a function to set options for the Issuer
type IssuerOption func(i *Issuer) error

// WithRSAPrivateKey sets the private RSA for the Issuer
func WithRSAPrivateKey(key *rsa.PrivateKey) IssuerOption {
	return func(i *Issuer) error {
		i.privateKey = key
		return nil
	}
}

func WithRSAPublicKey(key *rsa.PublicKey) IssuerOption {
	return func(i *Issuer) error {
		i.publicKey = key
		return nil
	}
}

// WithRSAPrivateKeyFromFile loads a PEM-encoded RSA private key from path and
// sets it as the private RSA key for the Issuer
func WithRSAPrivateKeyFromFile(path string) IssuerOption {
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
			return fmt.Errorf("no RSA private key in %s: %w", path, err)
		}
		i.privateKey = key
		return nil
	}
}

// WithRSAPublicKeyFromFile loads a PEM-encoded RSA private key from path and
// sets it as the private RSA key for the Issuer
func WithRSAPublicKeyFromFile(path string) IssuerOption {
	return func(i *Issuer) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read RSA public key: %w", err)
		}
		p, _ := pem.Decode(data)
		if p == nil {
			return fmt.Errorf("no valid PEM data found in %s", path)
		}
		key, err := x509.ParsePKCS1PublicKey(p.Bytes)
		if err != nil {
			return fmt.Errorf("no RSA public key in %s: %w", path, err)
		}
		i.publicKey = key
		return nil
	}
}

// NewIssuer creates a new instance of Issuer, which is used to issue JWTs
// to authenticated clients and to validate incoming JWTs.
func NewIssuer(name string, opts ...IssuerOption) (*Issuer, error) {
	iss := &Issuer{
		name:       name,
		atAudience: name + "-access",
		rtAudience: name + "-refresh",
	}
	for _, o := range opts {
		if err := o(iss); err != nil {
			return nil, err
		}
	}
	return iss, nil
}

func (i *Issuer) validationKey(t *jwt.Token) (interface{}, error) {
	var pubKey *rsa.PublicKey
	switch t.Method {
	case jwt.SigningMethodRS512:
		if i.publicKey != nil {
			pubKey = i.publicKey
		} else {
			pubKey = &i.privateKey.PublicKey
		}
	default:
		return nil, fmt.Errorf("token isn't signed with %s method", jwt.SigningMethodRS512)
	}

	return pubKey, nil
}

func (i *Issuer) validateToken(token string, aud string) (jwt.Claims, error) {
	t, err := jwt.Parse(token, i.validationKey,
		jwt.WithAudience(aud),
		jwt.WithIssuer(i.name),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS512.Name}),
	)
	if err != nil {
		return nil, fmt.Errorf("could not validate token: %w", err)
	}

	return t.Claims, nil
}

// IssueAccessToken creates and signs a new refresh token for client, which is
// valid for the duration specified as exp. The result is returned as a string.
func (i *Issuer) IssueAccessToken(client string, exp time.Duration) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		Issuer:    i.name,
		Subject:   client,
		Audience:  jwt.ClaimStrings{i.atAudience},
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
	})
	return t.SignedString(i.privateKey)
}

// IssueRefreshToken creates and signs a new refresh token for client, which is
// valid for the duration specified as exp. The result is returned as a string.
func (i *Issuer) IssueRefreshToken(client string, exp time.Duration) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		Issuer:    i.name,
		Subject:   client,
		Audience:  jwt.ClaimStrings{i.rtAudience},
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
	})
	return t.SignedString(i.privateKey)
}

// ValidateAccessToken validates an access token. On successfull validation,
// it returns the claims from the token. If validation fails, an error with
// the failure reason is returned.
func (i *Issuer) ValidateAccessToken(token string) (jwt.Claims, error) {
	return i.validateToken(token, i.atAudience)
}

// ValidateRefreshToken validates an access token. On successfull validation,
// it returns the claims from the token. If validation fails, an error with
// the failure reason is returned.
func (i *Issuer) ValidateRefreshToken(token string) (jwt.Claims, error) {
	return i.validateToken(token, i.rtAudience)
}
