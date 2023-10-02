package cache

import (
	"errors"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

var ErrCacheMiss = errors.New("entry does not exist in cache")

type AppCache interface {
	Get(key string) (*argo.Application, error)
	Store(key string, app *argo.Application) error
	Clear() error
}
