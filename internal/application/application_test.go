package application

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jannfis/argocd-agent/internal/appinformer"
	"github.com/jannfis/argocd-agent/internal/backend/kubernetes"
	appmock "github.com/jannfis/argocd-agent/internal/backend/mocks"
	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var appExistsError = errors.NewAlreadyExists(schema.GroupResource{Group: "argoproj.io", Resource: "application"}, "existing")
var appNotFoundError = errors.NewNotFound(schema.GroupResource{Group: "argoproj.io", Resource: "application"}, "existing")

func Test_ManagerOptions(t *testing.T) {
	t.Run("NewManager with default options", func(t *testing.T) {
		m := NewManager(nil)
		assert.Equal(t, false, m.AllowUpsert)
		assert.Nil(t, m.Metrics)
	})

	t.Run("NewManager with metrics", func(t *testing.T) {
		m := NewManager(nil, WithMetrics(metrics.NewApplicationClientMetrics()))
		assert.NotNil(t, m.Metrics)
	})

	t.Run("NewManager with upsert enabled", func(t *testing.T) {
		m := NewManager(nil, WithAllowUpsert(true))
		assert.True(t, m.AllowUpsert)
	})
}

func Test_ManagerCreate(t *testing.T) {
	t.Run("Create an application that exists", func(t *testing.T) {
		mockedBackend := appmock.NewApplication(t)
		mockedBackend.On("Create", mock.Anything, mock.Anything, mock.Anything).
			Return(func(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
				if app.Name == "existing" {
					return nil, appExistsError
				} else {
					return nil, nil
				}
			})
		m := NewManager(mockedBackend)
		_, err := m.Create(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.ErrorIs(t, err, appExistsError)
	})

	t.Run("Create a new application", func(t *testing.T) {
		app := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}
		mockedBackend := appmock.NewApplication(t)
		m := NewManager(mockedBackend)
		mockedBackend.On("Create", mock.Anything, mock.Anything).Return(app, nil)
		rapp, err := m.Create(context.TODO(), app)
		assert.NoError(t, err)
		assert.Equal(t, "test", rapp.Name)
	})
}

func prettyPrint(app *v1alpha1.Application) {
	b, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", b)
}

func Test_ManagerUpdateStatus_Fake(t *testing.T) {
	t.Run("Update status", func(t *testing.T) {
		incoming := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foobar",
				Namespace: "argocd",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"bar": "foo",
				},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        "github.com",
					TargetRevision: "HEAD",
					Path:           ".",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "in-cluster",
					Namespace: "guestbook",
				},
			},
		}
		existing := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foobar",
				Namespace: "argocd",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"bar":                        "foo",
					"argocd.argoproj.io/refresh": "normal",
				},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        "github.com",
					TargetRevision: "HEAD",
					Path:           ".",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "in-cluster",
					Namespace: "guestbook",
				},
			},
			Operation: &v1alpha1.Operation{
				InitiatedBy: v1alpha1.OperationInitiator{Username: "hello"},
			},
		}

		appC := fakeappclient.NewSimpleClientset(existing)
		informer := appinformer.NewAppInformer(context.Background(), appC, "argocd")
		be := kubernetes.NewKubernetesBackend(appC, informer, true)
		mgr := NewManager(be)
		updated, err := mgr.UpdateAutonomous(context.TODO(), incoming)
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.NotContains(t, updated.ObjectMeta.Annotations, "argocd.argoproj.io/refresh")
		require.Equal(t, map[string]string{"foo": "bar"}, updated.Labels)
	})
}

func Test_ManagerUpdateOperation_Fake(t *testing.T) {
	t.Run("Update status", func(t *testing.T) {
		incoming := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foobar",
				Namespace: "argocd",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"argocd.argoproj.io/refresh": "normal",
				},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        "github.com",
					TargetRevision: "HEAD",
					Path:           ".",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "in-cluster",
					Namespace: "guestbook",
				},
			},
			Operation: &v1alpha1.Operation{
				InitiatedBy: v1alpha1.OperationInitiator{Username: "hello"},
			},
		}
		existing := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foobar",
				Namespace: "argocd",
				Labels: map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL:        "github.com",
					TargetRevision: "HEAD",
					Path:           ".",
				},
				Destination: v1alpha1.ApplicationDestination{
					Server:    "in-cluster",
					Namespace: "guestbook",
				},
			},
			Operation: &v1alpha1.Operation{
				InitiatedBy: v1alpha1.OperationInitiator{Username: "foobar"},
			},
		}

		appC := fakeappclient.NewSimpleClientset(existing)
		informer := appinformer.NewAppInformer(context.Background(), appC, "argocd")
		be := kubernetes.NewKubernetesBackend(appC, informer, true)
		mgr := NewManager(be)
		updated, err := mgr.UpdateOperation(context.TODO(), incoming)
		require.NoError(t, err)
		require.NotNil(t, updated)
		prettyPrint(updated)
	})
}

func Test_ManagerUpdateStatus_Mocked(t *testing.T) {
	app := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "existing",
			Namespace: "default",
		},
	}
	t.Run("Update existing application", func(t *testing.T) {
		existing := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "existing",
				Namespace: "default",
				Labels: map[string]string{
					"foo": "bar",
				},
				Annotations: map[string]string{
					"bar": "foo",
				},
			},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL: "foo",
				},
			},
			Status: v1alpha1.ApplicationStatus{
				Sync: v1alpha1.SyncStatus{
					Status: v1alpha1.SyncStatusCodeOutOfSync,
				},
			},
		}
		incoming := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "existing",
				Namespace: "default",
				Labels: map[string]string{
					"foo": "bar",
					"bar": "foo",
				},
				Annotations: map[string]string{
					"foo": "bar",
				},
				ResourceVersion: "3",
			},
			Operation: &v1alpha1.Operation{},
			Spec: v1alpha1.ApplicationSpec{
				Source: &v1alpha1.ApplicationSource{
					RepoURL: "bar",
				},
			},
			Status: v1alpha1.ApplicationStatus{
				Sync: v1alpha1.SyncStatus{
					Status: v1alpha1.SyncStatusCodeSynced,
				},
			},
		}

		mockedBackend := appmock.NewApplication(t)
		m := NewManager(mockedBackend)
		mockedBackend.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(existing, nil)
		mockedBackend.On("SupportsPatch").Return(true)
		mockedBackend.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(incoming, nil)
		napp, err := m.UpdateAutonomous(context.TODO(), incoming)
		assert.NoError(t, err)
		assert.Equal(t, napp, incoming)
		assert.True(t, m.IsChangeIgnored(napp.QualifiedName(), napp.ResourceVersion))
	})

	t.Run("Update non-existing application", func(t *testing.T) {
		mockedBackend := appmock.NewApplication(t)
		m := NewManager(mockedBackend)
		mockedBackend.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil, appNotFoundError)
		_, err := m.UpdateAutonomous(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.ErrorIs(t, err, appNotFoundError)
	})

	t.Run("Upsert non-existing application", func(t *testing.T) {
		mockedBackend := appmock.NewApplication(t)
		m := NewManager(mockedBackend, WithAllowUpsert(true))
		mockedBackend.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(nil, appNotFoundError)
		mockedBackend.On("Create", mock.Anything, mock.Anything).Return(app, nil)
		napp, err := m.UpdateAutonomous(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.NoError(t, err)
		assert.Equal(t, app, napp)
	})
}

func Test_ManageApp(t *testing.T) {
	t.Run("Mark app as managed", func(t *testing.T) {
		appm := NewManager(nil)
		assert.False(t, appm.IsManaged("foo"))
		err := appm.Manage("foo")
		assert.NoError(t, err)
		assert.True(t, appm.IsManaged("foo"))
		err = appm.Manage("foo")
		assert.Error(t, err)
		assert.True(t, appm.IsManaged("foo"))
		appm.ClearManaged()
		assert.False(t, appm.IsManaged("foo"))
		assert.Len(t, appm.managedApps, 0)
	})

	t.Run("Mark app as unmanaged", func(t *testing.T) {
		appm := NewManager(nil)
		err := appm.Manage("foo")
		assert.True(t, appm.IsManaged("foo"))
		assert.NoError(t, err)
		err = appm.Unmanage("foo")
		assert.NoError(t, err)
		assert.False(t, appm.IsManaged("foo"))
		err = appm.Unmanage("foo")
		assert.Error(t, err)
		assert.False(t, appm.IsManaged("foo"))
	})
}

func Test_IgnoreChange(t *testing.T) {
	t.Run("Ignore a change", func(t *testing.T) {
		appm := NewManager(nil)
		assert.False(t, appm.IsChangeIgnored("foo", "1"))
		err := appm.IgnoreChange("foo", "1")
		assert.NoError(t, err)
		assert.True(t, appm.IsChangeIgnored("foo", "1"))
		err = appm.IgnoreChange("foo", "1")
		assert.Error(t, err)
		assert.True(t, appm.IsChangeIgnored("foo", "1"))
		appm.ClearIgnored()
		assert.False(t, appm.IsChangeIgnored("foo", "1"))
		assert.Len(t, appm.managedApps, 0)
	})

	t.Run("Unignore a change", func(t *testing.T) {
		appm := NewManager(nil)
		err := appm.UnignoreChange("foo")
		assert.Error(t, err)
		assert.False(t, appm.IsChangeIgnored("foo", "1"))
		err = appm.IgnoreChange("foo", "1")
		assert.NoError(t, err)
		assert.True(t, appm.IsChangeIgnored("foo", "1"))
		err = appm.UnignoreChange("foo")
		assert.NoError(t, err)
		assert.False(t, appm.IsChangeIgnored("foo", "1"))
		err = appm.UnignoreChange("foo")
		assert.Error(t, err)
		assert.False(t, appm.IsChangeIgnored("foo", "1"))
	})
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
