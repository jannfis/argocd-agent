package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockAuth struct{}

func (a *mockAuth) Authenticate(creds Credentials) (bool, error) {
	return true, nil
}

func Test_AuthMethods(t *testing.T) {
	t.Run("Register an auth method and verify", func(t *testing.T) {
		m := NewMethods()
		authmethod := &mockAuth{}
		err := m.RegisterAuthMethod("userpass", authmethod)
		require.NoError(t, err)
		require.NotNil(t, m.AuthMethod("userpass"))
	})

	t.Run("Register two auth methods under same name", func(t *testing.T) {
		m := NewMethods()
		authmethod := &mockAuth{}
		err := m.RegisterAuthMethod("userpass", authmethod)
		require.NoError(t, err)
		err = m.RegisterAuthMethod("userpass", authmethod)
		require.Error(t, err)
	})

	t.Run("Look up non-existing auth method", func(t *testing.T) {
		m := NewMethods()
		authmethod := &mockAuth{}
		err := m.RegisterAuthMethod("userpass", authmethod)
		require.NoError(t, err)
		require.NotNil(t, m.AuthMethod("userpass"))
		require.Nil(t, m.AuthMethod("username"))
	})

}
