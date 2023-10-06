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
		auths := NewServer(nil)
		_, err := auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{"username": "user1", "password": "password"}},
		)
		assert.ErrorContains(t, err, "unsupported authentication method")
	})
	t.Run("Authentication successful", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterAuthMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams)
		r, err := auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{"username": "user1", "password": "password"}},
		)
		require.NoError(t, err)
		assert.Equal(t, types.AuthResultOK, r.Result)
	})

	t.Run("Wrong credentials", func(t *testing.T) {
		ams := auth.NewMethods()
		am := userpass.NewUserPassAuthentication()
		am.UpsertUser("user1", "password")
		err := ams.RegisterAuthMethod("userpass", am)
		require.NoError(t, err)
		auths := NewServer(ams)
		_, err = auths.Authenticate(context.TODO(), &authapi.AuthRequest{
			Method:      "userpass",
			Credentials: map[string]string{"username": "user1", "password": "wordpass"}},
		)
		require.ErrorContains(t, err, "authentication failed")
	})

}
