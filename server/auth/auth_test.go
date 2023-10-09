package auth

import (
	"context"
	"testing"

	"github.com/jannfis/argocd-application-agent/internal/auth"
	"github.com/jannfis/argocd-application-agent/internal/auth/userpass"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/authapi"
	"github.com/jannfis/argocd-application-agent/pkg/types"
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
		assert.ErrorContains(t, err, "unsupported authentication method")
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
		assert.Equal(t, types.AuthResultOK, r.Result)
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
