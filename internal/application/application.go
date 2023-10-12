package application

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/jannfis/argocd-agent/server/backend"
	"github.com/sirupsen/logrus"
)

// Manager manages Argo CD application resources on a given backend.
//
// It provides primitives to create, update, upsert and delete applications.
type Manager struct {
	AllowUpsert bool
	Backend     backend.Application
	Metrics     *metrics.ApplicationClientMetrics
}

type ManagerOption func(*Manager)

// WithMetrics sets the metrics provider for the Manager
func WithMetrics(m *metrics.ApplicationClientMetrics) ManagerOption {
	return func(mgr *Manager) {
		mgr.Metrics = m
	}
}

// WithAllowUpsert sets the upsert operations allowed flag
func WithAllowUpsert(upsert bool) ManagerOption {
	return func(m *Manager) {
		m.AllowUpsert = upsert
	}
}

// NewManager initializes and returns a new Manager with the given backend and
// options.
func NewManager(be backend.Application, opts ...ManagerOption) *Manager {
	m := &Manager{}
	for _, o := range opts {
		o(m)
	}
	m.Backend = be
	return m
}

// Create creates the application app using the Manager's application backend.
func (m *Manager) Create(ctx context.Context, app *v1alpha1.Application) error {
	_, err := m.Backend.Create(ctx, app)
	if err == nil {
		if m.Metrics != nil {
			m.Metrics.AppsCreated.WithLabelValues(app.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}
	return err
}

// UpdateStatus updates the status field of an Application. This method is
// usually executed when the agent submits the status update. If the app
// does not yet exist, and the Manager m allows upsert, the app is created
// on the control plane's application backend.
func (m *Manager) UpdateStatus(ctx context.Context, app *v1alpha1.Application) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		logCtx := log().WithField("component", "UpdateStatus")
		exApp, err := m.Backend.Get(ctx, app.Name, app.Namespace)
		if err != nil {
			if errors.IsNotFound(err) && m.AllowUpsert {
				logCtx.WithField("application", app.QualifiedName()).Infof("Creating application")
				return m.Create(ctx, app)
			} else {
				return fmt.Errorf("could not get app %s: %w", app.QualifiedName(), err)
			}
		} else {
			exApp.Status = *app.Status.DeepCopy()
			_, err = m.Backend.Update(ctx, exApp)
		}
		return err
	})
	if err == nil {
		if m.Metrics != nil {
			m.Metrics.AppsUpdated.WithLabelValues(app.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}
	return err
}

func log() *logrus.Entry {
	return logrus.WithField("component", "AppManager")
}
