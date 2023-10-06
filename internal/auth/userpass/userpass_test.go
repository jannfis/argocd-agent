package userpass

import (
	"testing"

	"github.com/jannfis/argocd-application-agent/internal/auth"
	"github.com/stretchr/testify/assert"
)

func Test_UpsertUser(t *testing.T) {
	a := NewUserPassAuthentication()
	assert.Len(t, a.userdb, 0)
	t.Run("Add a new user", func(t *testing.T) {
		a.UpsertUser("user1", "password")
		assert.Len(t, a.userdb, 1)
		assert.Contains(t, a.userdb, "user1")
		assert.Equal(t, "password", a.userdb["user1"])
	})
	t.Run("Add another new user", func(t *testing.T) {
		a.UpsertUser("user2", "password")
		assert.Len(t, a.userdb, 2)
		assert.Contains(t, a.userdb, "user1")
		assert.Contains(t, a.userdb, "user2")
		assert.Equal(t, "password", a.userdb["user1"])
		assert.Equal(t, "password", a.userdb["user2"])
	})
	t.Run("Update existing user", func(t *testing.T) {
		a.UpsertUser("user1", "wordpass")
		assert.Len(t, a.userdb, 2)
		assert.Contains(t, a.userdb, "user1")
		assert.Contains(t, a.userdb, "user2")
		assert.Equal(t, "wordpass", a.userdb["user1"])
		assert.Equal(t, "password", a.userdb["user2"])
	})
}

func Test_Authenticate(t *testing.T) {
	a := NewUserPassAuthentication()
	t.Run("Successful authentication", func(t *testing.T) {
		a.UpsertUser("user1", "password")
		creds := make(auth.Credentials)
		creds["username"] = "user1"
		creds["password"] = "password"
		ok, err := a.Authenticate(creds)
		assert.True(t, ok)
		assert.NoError(t, err)
	})

	t.Run("Unknown user", func(t *testing.T) {
		a.UpsertUser("user1", "password")
		creds := make(auth.Credentials)
		creds["username"] = "user2"
		creds["password"] = "password"
		ok, err := a.Authenticate(creds)
		assert.False(t, ok)
		assert.Error(t, err)
	})

	t.Run("Wrong password", func(t *testing.T) {
		creds := make(auth.Credentials)
		creds["username"] = "user1"
		creds["password"] = "wordpass"
		ok, err := a.Authenticate(creds)
		assert.False(t, ok)
		assert.Error(t, err)
	})

	t.Run("Missing password", func(t *testing.T) {
		creds := make(auth.Credentials)
		creds["username"] = "user1"
		ok, err := a.Authenticate(creds)
		assert.False(t, ok)
		assert.Error(t, err)
	})

	t.Run("Missing username", func(t *testing.T) {
		creds := make(auth.Credentials)
		creds["password"] = "password"
		ok, err := a.Authenticate(creds)
		assert.False(t, ok)
		assert.Error(t, err)
	})

}
