package informer

import (
	"time"

	"github.com/jannfis/argocd-agent/internal/filter"
	"github.com/jannfis/argocd-agent/internal/metrics"
)

// InformerOptions is a set of options for the AppInformer.
//
// Options should not be modified concurrently, they are not implemented in a
// thread-safe way.
type InformerOptions struct {
	namespace  string
	namespaces []string
	appMetrics *metrics.ApplicationWatcherMetrics
	filters    *filter.Chain
	resync     time.Duration
	listCb     ListAppsCallback
	newCb      NewAppCallback
	updateCb   UpdateAppCallback
	deleteCb   DeleteAppCallback
	errorCb    ErrorCallback
}

type InformerOption func(o *InformerOptions)

// WithMetrics sets the ApplicationMetrics instance to be used by the AppInformer
func WithMetrics(m *metrics.ApplicationWatcherMetrics) InformerOption {
	return func(o *InformerOptions) {
		o.appMetrics = m
	}
}

// WithNamespaces sets additional namespaces to be watched by the AppInformer
func WithNamespaces(namespaces ...string) InformerOption {
	return func(o *InformerOptions) {
		o.namespaces = namespaces
	}
}

// WithFilterChain sets the FilterChain to be used by the AppInformer
func WithFilterChain(fc *filter.Chain) InformerOption {
	return func(o *InformerOptions) {
		o.filters = fc
	}
}

// WithListAppCallback sets the ListAppsCallback to be called by the AppInformer
func WithListAppCallback(cb ListAppsCallback) InformerOption {
	return func(o *InformerOptions) {
		o.listCb = cb
	}
}

// WithNewAppCallback sets the NewAppCallback to be executed by the AppInformer
func WithNewAppCallback(cb NewAppCallback) InformerOption {
	return func(o *InformerOptions) {
		o.newCb = cb
	}
}

// WithUpdateAppCallback sets the UpdateAppCallback to be executed by the AppInformer
func WithUpdateAppCallback(cb UpdateAppCallback) InformerOption {
	return func(o *InformerOptions) {
		o.updateCb = cb
	}
}

// WithDeleteAppCallback sets the DeleteAppCallback to be executed by the AppInformer
func WithDeleteAppCallback(cb DeleteAppCallback) InformerOption {
	return func(o *InformerOptions) {
		o.deleteCb = cb
	}
}

// WithErrorCallback sets the ErrorCallback to be executed by the AppInformer
func WithErrorCallback(cb ErrorCallback) InformerOption {
	return func(o *InformerOptions) {
		o.errorCb = cb
	}
}

// WithResyncDuration sets the resync duration to be used by the AppInformer's lister
func WithResyncDuration(d time.Duration) InformerOption {
	return func(o *InformerOptions) {
		o.resync = d
	}
}
