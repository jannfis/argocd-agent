package userpass

import (
	"fmt"
	"sync"

	"github.com/jannfis/argocd-application-agent/internal/auth"
)

type userPassAuthentication struct {
	lock   sync.RWMutex
	userdb map[string]string
}

func NewUserPassAuthentication() *userPassAuthentication {
	return &userPassAuthentication{
		userdb: make(map[string]string),
	}
}

func (a *userPassAuthentication) Authenticate(creds auth.Credentials) (authenticated bool, autherr error) {
	username, ok := creds["username"]
	if !ok {
		return false, fmt.Errorf("username is missing from credentials")
	}
	password, ok := creds["password"]
	if !ok {
		return false, fmt.Errorf("password is missing from credentials")
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	pass, ok := a.userdb[username]
	if !ok {
		return false, fmt.Errorf("user not found: %s", username)
	}

	if pass == password {
		return true, nil
	}

	return false, fmt.Errorf("authentication failed")
}

func (a *userPassAuthentication) UpsertUser(username, password string) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.userdb[username] = password
}
