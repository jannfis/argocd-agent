package auth

// Credentials is a data type for passing arbitrary credentials to auth methods
type Credentials map[string]string

// Method is the interface to be implemented by all auth methods
type Method interface {
	Authenticate(credentials Credentials) (bool, error)
}
