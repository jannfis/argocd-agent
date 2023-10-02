package agent

func WithAllowedNamespaces(namespaces ...string) AgentOption {
	return func(co *AgentOptions) {
		co.namespaces = namespaces
	}
}

func WithAgentNamespace(namespace string) AgentOption {
	return func(co *AgentOptions) {
		co.namespace = namespace
	}
}
