package auth

import (
	"context"
	"testing"

	"github.com/jannfis/argocd-agent/internal/auth"
	"github.com/jannfis/argocd-agent/internal/auth/userpass"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/authapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Authenticate(t *testing.T) {
	t.Run("Authentication method unsupported", func(t *testing.T) {
		auths := NewServer(nil, nil)
		_, err := auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{userpass.ClientIDField: "user1", userpass.ClientSecretField: "password"}},
		)
		assert.ErrorContains(t, err, authFailedMessage)
	})
	t.Run("Authentication successful", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams, nil)
		r, err := auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{userpass.ClientIDField: "user1", userpass.ClientSecretField: "password"}},
		)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.NotEmpty(t, r.AccessToken)
		assert.NotEmpty(t, r.RefreshToken)
		claims, err := auths.issuer.ValidateAccessToken(r.AccessToken)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		userid, err := claims.GetSubject()
		assert.NoError(t, err)
		assert.Equal(t, "user1", userid)
		claims, err = auths.issuer.ValidateRefreshToken(r.RefreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		userid, err = claims.GetSubject()
		assert.NoError(t, err)
		assert.Equal(t, "user1", userid)

	})

	t.Run("Wrong credentials", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams, nil)
		_, err = auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{userpass.ClientIDField: "user1", userpass.ClientSecretField: "wordpass"}},
		)
		require.ErrorContains(t, err, "authentication failed")
	})
	t.Run("Incomplete credentials", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams, nil)
		_, err = auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{"foo": "bar"}},
		)
		require.ErrorContains(t, err, "authentication failed")
	})

}

func Test_RefreshToken(t *testing.T) {
	t.Run("Request a refresh token", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams, nil)
		r, err := auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{userpass.ClientIDField: "user1", userpass.ClientSecretField: "password"}},
		)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.NotEmpty(t, r.AccessToken)
		assert.NotEmpty(t, r.RefreshToken)
		nr, err := auths.RefreshToken(context.TODO(), &authapi.RefreshTokenRequest{RefreshToken: r.RefreshToken})
		require.NoError(t, err)
		require.NotNil(t, nr)
		assert.NotEqual(t, r.AccessToken, nr.AccessToken)
		assert.NotEqual(t, r.RefreshToken, nr.RefreshToken)
	})
}
