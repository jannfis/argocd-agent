package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/backend"
	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/sirupsen/logrus"
	"github.com/wI2L/jsondiff"
)

type ManagerLocation int
type ManagerMode int

type updateTransformer func(existing, incoming *v1alpha1.Application)
type patchTransformer func(existing, incoming *v1alpha1.Application) (jsondiff.Patch, error)

const (
	ManagerLocationUnset ManagerLocation = iota
	ManagerLocationServer
	ManagerLocationAgent
)

const (
	ManagerModeUnset ManagerMode = iota
	ManagerModeAutonomous
	ManagerModeManaged
)

// Manager manages Argo CD application resources on a given backend.
//
// It provides primitives to create, update, upsert and delete applications.
type Manager struct {
	AllowUpsert bool
	Application backend.Application
	Metrics     *metrics.ApplicationClientMetrics
	Location    ManagerLocation
	Mode        ManagerMode
	managedApps map[string]bool // Managed apps is a list of apps we manage
	observedApp map[string]string
	lock        sync.RWMutex
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

func WithLocation(loc ManagerLocation) ManagerOption {
	return func(m *Manager) {
		m.Location = loc
	}
}

func WithMode(mode ManagerMode) ManagerOption {
	return func(m *Manager) {
		m.Mode = mode
	}
}

// NewManager initializes and returns a new Manager with the given backend and
// options.
func NewManager(be backend.Application, opts ...ManagerOption) *Manager {
	m := &Manager{}
	for _, o := range opts {
		o(m)
	}
	m.Application = be
	m.observedApp = make(map[string]string)
	m.managedApps = make(map[string]bool)
	return m
}

// Create creates the application app using the Manager's application backend.
func (m *Manager) Create(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {

	// A new Application must neither specify ResourceVersion nor Generation
	app.ResourceVersion = ""
	app.Generation = 0

	// We never want Operation to be set on the server side.
	if m.Location == ManagerLocationServer {
		app.Operation = nil
	}

	created, err := m.Application.Create(ctx, app)
	if err == nil {
		m.Manage(created.QualifiedName())
		m.IgnoreChange(created.QualifiedName(), created.ResourceVersion)
		if m.Metrics != nil {
			m.Metrics.AppsCreated.WithLabelValues(app.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}

	return created, err
}

func (m *Manager) UpdateManaged(ctx context.Context, incoming *v1alpha1.Application) (*v1alpha1.Application, error) {
	logCtx := log().WithFields(logrus.Fields{
		"component":       "UpdateSpec",
		"application":     incoming.QualifiedName(),
		"resourceVersion": incoming.ResourceVersion,
	})

	var updated *v1alpha1.Application
	var err error
	updated, err = m.update(ctx, incoming, func(existing, incoming *v1alpha1.Application) {
		existing.ObjectMeta.Annotations = incoming.ObjectMeta.Annotations
		existing.ObjectMeta.Labels = incoming.ObjectMeta.Labels
		existing.Spec = *incoming.Spec.DeepCopy()
		existing.Operation = incoming.Operation.DeepCopy()
		existing.Status = *incoming.Status.DeepCopy()
	}, func(existing, incoming *v1alpha1.Application) (jsondiff.Patch, error) {
		target := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Labels:      incoming.Labels,
				Annotations: incoming.Labels,
			},
			Spec: incoming.Spec,
		}
		source := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Labels:      existing.Labels,
				Annotations: existing.Annotations,
			},
			Spec: existing.Spec,
		}
		return jsondiff.Compare(source, target)
	})
	if err == nil {
		if updated.Generation == 1 {
			logCtx.Infof("Created application")
		} else {
			logCtx.Infof("Updated application")
		}
		m.IgnoreChange(updated.QualifiedName(), updated.ResourceVersion)
		if m.Metrics != nil {
			m.Metrics.AppsUpdated.WithLabelValues(incoming.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}
	return updated, err
}

// UpdateAutonomous updates the Application resource on the control plane side
// when the agent is in autonomous mode. It will update changes to .spec and
// .status fields along with syncing labels and annotations.
//
// Additionally, it will remove any .operation field from the incoming resource
// before the resource is being updated on the control plane.
//
// This method is usually only executed by the control plane for updates that
// are received by agents in autonomous mode.
func (m *Manager) UpdateAutonomous(ctx context.Context, incoming *v1alpha1.Application) (*v1alpha1.Application, error) {
	logCtx := log().WithFields(logrus.Fields{
		"component":       "UpdateStatus",
		"application":     incoming.QualifiedName(),
		"resourceVersion": incoming.ResourceVersion,
	})

	var updated *v1alpha1.Application
	var err error
	updated, err = m.update(ctx, incoming, func(existing, incoming *v1alpha1.Application) {
		existing.ObjectMeta.Annotations = incoming.ObjectMeta.Annotations
		existing.ObjectMeta.Labels = incoming.ObjectMeta.Labels
		existing.Spec = incoming.Spec
		existing.Status = *incoming.Status.DeepCopy()
		existing.Operation = nil
		logCtx.Infof("Updating")
	}, func(existing, incoming *v1alpha1.Application) (jsondiff.Patch, error) {
		target := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Labels:      incoming.Labels,
				Annotations: incoming.Annotations,
			},
			Spec:   incoming.Spec,
			Status: incoming.Status,
		}
		source := &v1alpha1.Application{
			Spec:   existing.Spec,
			Status: existing.Status,
		}
		patch, err := jsondiff.Compare(source, target)
		if err != nil {
			return nil, err
		}

		// Append remove operation for operation field if it exists. We neither
		// want nor need it on the control plane's resource.
		if existing.Operation != nil {
			patch = append(patch, jsondiff.Operation{Type: "remove", Path: "/operation"})
		}
		logCtx.Tracef("Patching: %s", patch)
		return patch, nil
	})
	if err == nil {
		m.IgnoreChange(updated.QualifiedName(), updated.ResourceVersion)
		logCtx.WithField("newResourceVersion", updated.ResourceVersion).Infof("Updated application status")
		if m.Metrics != nil {
			m.Metrics.AppsUpdated.WithLabelValues(incoming.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}
	return updated, err
}

// UpdateOperation is used to update the .operation field of the application
// resource to initiate a sync. Additionally, any labels and annotations that
// are used to trigger an action (such as, refresh) will be set on the target
// resource.
//
// This method is usually executed only by an agent in autonomous mode, because
// it has the leading version of the resource and we are not supposed to change
// its Application manifests.
func (m *Manager) UpdateOperation(ctx context.Context, incoming *v1alpha1.Application) (*v1alpha1.Application, error) {
	logCtx := log().WithFields(logrus.Fields{
		"component":       "UpdateOperation",
		"application":     incoming.QualifiedName(),
		"resourceVersion": incoming.ResourceVersion,
	})

	var updated *v1alpha1.Application
	var err error
	updated, err = m.update(ctx, incoming, func(existing, incoming *v1alpha1.Application) {
		existing.ObjectMeta.Annotations = incoming.ObjectMeta.Annotations
		existing.ObjectMeta.Labels = incoming.ObjectMeta.Labels
		existing.Status = *incoming.Status.DeepCopy()
	}, func(existing, incoming *v1alpha1.Application) (jsondiff.Patch, error) {
		annotations := make(map[string]string)
		for k, v := range incoming.Annotations {
			if k != "argocd.argoproj.io/refresh" {
				annotations[k] = v
			}
		}
		target := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Annotations: incoming.Annotations,
			},
			Operation: incoming.Operation,
		}
		source := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Annotations: annotations,
			},
			Operation: existing.Operation,
		}
		patch, err := jsondiff.Compare(source, target, jsondiff.SkipCompact())
		return patch, err
	})
	if err == nil {
		m.IgnoreChange(updated.QualifiedName(), updated.ResourceVersion)
		logCtx.WithField("newResourceVersion", updated.ResourceVersion).Infof("Updated application status")
		if m.Metrics != nil {
			m.Metrics.AppsUpdated.WithLabelValues(incoming.Namespace).Inc()
		}
	} else {
		if m.Metrics != nil {
			m.Metrics.Errors.Inc()
		}
	}
	return updated, err
}

func (m *Manager) update(ctx context.Context, incoming *v1alpha1.Application, updateFn updateTransformer, patchFn patchTransformer) (*v1alpha1.Application, error) {
	var updated *v1alpha1.Application
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		existing, ierr := m.Application.Get(ctx, incoming.Name, incoming.Namespace)
		if ierr != nil {
			if errors.IsNotFound(ierr) && m.AllowUpsert {
				updated, ierr = m.Create(ctx, incoming)
				return ierr
			} else {
				return fmt.Errorf("error updating application %s: %w", incoming.QualifiedName(), ierr)
			}
		} else {
			if m.Application.SupportsPatch() && patchFn != nil {
				patch, err := patchFn(existing, incoming)
				if err != nil {
					return fmt.Errorf("could not create patch: %w", err)
				}
				jsonpatch, err := json.Marshal(patch)
				if err != nil {
					return fmt.Errorf("could not marshal jsonpatch: %w", err)
				}
				updated, ierr = m.Application.Patch(ctx, incoming.Name, incoming.Namespace, jsonpatch)
			} else {
				if updateFn != nil {
					updateFn(existing, incoming)
				}
				updated, ierr = m.Application.Update(ctx, existing)
			}
		}
		return ierr
	})
	return updated, err
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
	m.observedApp = make(map[string]string)
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

// IgnoreChange adds a particular version for the application named appName to
// list of changes to ignore.
func (m *Manager) IgnoreChange(appName string, version string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if cur, ok := m.observedApp[appName]; ok && cur == version {
		return fmt.Errorf("version %s is already ignored for %s", version, appName)
	} else {
		m.observedApp[appName] = version
		return nil
	}
}

func (m *Manager) IsChangeIgnored(appName string, version string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	last, ok := m.observedApp[appName]
	if !ok {
		return false
	}
	return last == version
}

func (m *Manager) UnignoreChange(appName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.observedApp[appName]; ok {
		delete(m.observedApp, appName)
		return nil
	} else {
		return fmt.Errorf("no generation recorded for app %s", appName)
	}
}

func log() *logrus.Entry {
	return logrus.WithField("component", "AppManager")
}
