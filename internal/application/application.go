package application

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/backend"
	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/sirupsen/logrus"
)

// Manager manages Argo CD application resources on a given backend.
//
// It provides primitives to create, update, upsert and delete applications.
type Manager struct {
	AllowUpsert bool
	Backend     backend.Application
	Metrics     *metrics.ApplicationClientMetrics

	managedApps  map[string]bool            // Managed apps is a list of apps we manage
	ignoreChange map[string]map[string]bool // ignoreChange contains a list of app names and resource versions to ignore
	lock         sync.RWMutex
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
	m.ignoreChange = make(map[string]map[string]bool)
	m.managedApps = make(map[string]bool)
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
		exApp, ierr := m.Backend.Get(ctx, app.Name, app.Namespace)
		if ierr != nil {
			if errors.IsNotFound(ierr) && m.AllowUpsert {
				logCtx.WithField("application", app.QualifiedName()).Infof("Creating application")
				return m.Create(ctx, app)
			} else {
				return fmt.Errorf("could not get app %s: %w", app.QualifiedName(), ierr)
			}
		} else {
			exApp.Status = *app.Status.DeepCopy()
			var napp *v1alpha1.Application
			napp, ierr = m.Backend.Update(ctx, exApp)
			if ierr == nil {
				m.IgnoreChange(napp.QualifiedName(), napp.ResourceVersion)
			}
		}
		return ierr
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

// ClearManaged clears the managed apps
func (m *Manager) ClearManaged() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.managedApps = make(map[string]bool)
}

func (m *Manager) ClearIgnored() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.ignoreChange = make(map[string]map[string]bool)
}

// IsManaged returns whether the app appName is currently managed by this agent
func (m *Manager) IsManaged(appName string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.managedApps[appName]
	return ok
}

// Manage marks the app appName as being managed by this agent
func (m *Manager) Manage(appName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.managedApps[appName]
	if !ok {
		m.managedApps[appName] = true
		return nil
	} else {
		return fmt.Errorf("app %s is already managed", appName)
	}
}

// Unmanage marks the app appName as not being managed by this agent
func (m *Manager) Unmanage(appName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.managedApps[appName]
	if !ok {
		return fmt.Errorf("app %s is not managed", appName)
	} else {
		delete(m.managedApps, appName)
		return nil
	}
}

func (m *Manager) IgnoreChange(appName string, resourceVersion string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.ignoreChange[appName]
	if !ok {
		m.ignoreChange[appName] = make(map[string]bool)
	} else {
		return fmt.Errorf("change %s for app %s is already ignored", resourceVersion, appName)
	}
	m.ignoreChange[appName][resourceVersion] = true
	return nil
}

func (m *Manager) IsChangeIgnored(appName, resourceVersion string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.ignoreChange[appName]
	if !ok {
		return false
	}
	_, ok = m.ignoreChange[appName][resourceVersion]
	return ok
}

func (m *Manager) UnignoreChange(appName, resourceVersion string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.ignoreChange[appName]
	if ok {
		if _, ok = m.ignoreChange[appName][resourceVersion]; !ok {
			return fmt.Errorf("change %s for app %s is not ignored", resourceVersion, appName)
		}
		delete(m.ignoreChange[appName], resourceVersion)
		return nil
	} else {
		return fmt.Errorf("change %s for app %s is not ignored", resourceVersion, appName)
	}
}

func log() *logrus.Entry {
	return logrus.WithField("component", "AppManager")
}
