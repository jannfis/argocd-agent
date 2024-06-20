package manager

type ManagerRole int
type ManagerMode int

const (
	ManagerRoleUnset ManagerRole = iota
	ManagerRolePrincipal
	ManagerRoleAgent
)

const (
	ManagerModeUnset ManagerMode = iota
	ManagerModeAutonomous
	ManagerModeManaged
)

type Manager interface {
	SetRole(role ManagerRole)
	SetMode(role ManagerRole)
}

func (r ManagerRole) IsPrincipal() bool {
	return r == ManagerRolePrincipal
}

func (r ManagerRole) IsAgent() bool {
	return r == ManagerRoleAgent
}

func (m ManagerMode) IsAutonomous() bool {
	return m == ManagerModeAutonomous
}

func (m ManagerMode) IsManaged() bool {
	return m == ManagerModeManaged
}
