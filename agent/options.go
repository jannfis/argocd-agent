package agent

import "github.com/jannfis/argocd-agent/pkg/client"

func WithAllowedNamespaces(namespaces ...string) AgentOption {
	return func(a *Agent) error {
		a.allowedNamespaces = namespaces
		return nil
	}
}

func WithRemote(remote *client.Remote) AgentOption {
	return func(a *Agent) error {
		a.remote = remote
		return nil
	}
}
