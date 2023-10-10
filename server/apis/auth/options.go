package auth

import "github.com/jannfis/argocd-agent/internal/user"

func WithUserRegistry(r user.Registry) ServerOption {
	return func(o *ServerOptions) error {
		return nil
	}
}
