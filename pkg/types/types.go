package types

const (
	AuthResultOK           string = "ok"
	AuthResultUnauthorized string = "unauthorized"
)

type EventContextKey string

func (k EventContextKey) String() string {
	return string(k)
}

const ContextAgentIdentifier EventContextKey = "agent_name"
