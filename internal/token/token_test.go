package token

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signedTokenWithClaims(method jwt.SigningMethod, key interface{}, claims jwt.Claims) (string, error) {
	tok := jwt.NewWithClaims(method, claims)
	return tok.SignedString(key)
}

func Test_Issuer(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var tok string
	t.Run("Issue a JWT", func(t *testing.T) {
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		tok, err = i.Issue("agent", 5*time.Second)
		require.NoError(t, err)
	})
	t.Run("Valid JWT", func(t *testing.T) {
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		require.NoError(t, err)
		sub, err := c.GetSubject()
		require.NoError(t, err)
		assert.Equal(t, "agent", sub)
	})
	t.Run("JWT signed by another issuer", func(t *testing.T) {
		i1, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		i2, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		tok, err := i2.Issue("agent", 5*time.Second)
		require.NoError(t, err)
		c, err := i1.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrSignatureInvalid.Error())
		assert.Nil(t, c)
	})

	t.Run("JWT signed with forbidden none method", func(t *testing.T) {
		tok, err := signedTokenWithClaims(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.RegisteredClaims{
			Issuer:    "server",
			Subject:   "agent",
			Audience:  jwt.ClaimStrings{"server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		require.NoError(t, err)
		require.NotNil(t, tok)
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrSignatureInvalid.Error())
		assert.Nil(t, c)
	})

	t.Run("JWT with invalid audience", func(t *testing.T) {
		tok, err := signedTokenWithClaims(jwt.SigningMethodRS512, key, jwt.RegisteredClaims{
			Issuer:    "server",
			Subject:   "agent",
			Audience:  jwt.ClaimStrings{"agent"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		require.NoError(t, err)
		require.NotNil(t, tok)
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrTokenInvalidAudience.Error())
		assert.Nil(t, c)
	})

	t.Run("JWT with invalid issuer", func(t *testing.T) {
		tok, err := signedTokenWithClaims(jwt.SigningMethodRS512, key, jwt.RegisteredClaims{
			Issuer:    "agent",
			Subject:   "agent",
			Audience:  jwt.ClaimStrings{"server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		require.NoError(t, err)
		require.NotNil(t, tok)
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrTokenInvalidIssuer.Error())
		assert.Nil(t, c)
	})

	t.Run("Expired JWT", func(t *testing.T) {
		tok, err := signedTokenWithClaims(jwt.SigningMethodRS512, key, jwt.RegisteredClaims{
			Issuer:    "server",
			Subject:   "agent",
			Audience:  jwt.ClaimStrings{"server"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		require.NoError(t, err)
		require.NotNil(t, tok)
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrTokenExpired.Error())
		assert.Nil(t, c)
	})

	t.Run("JWT not yet valid", func(t *testing.T) {
		tok, err := signedTokenWithClaims(jwt.SigningMethodRS512, key, jwt.RegisteredClaims{
			Issuer:    "server",
			Subject:   "agent",
			Audience:  jwt.ClaimStrings{"server"},
			NotBefore: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		require.NoError(t, err)
		require.NotNil(t, tok)
		i, err := NewIssuer("server", WithPrivateRSAKey(key))
		require.NoError(t, err)
		c, err := i.Validate(tok)
		assert.ErrorContains(t, err, jwt.ErrTokenNotValidYet.Error())
		assert.Nil(t, c)
	})

}
