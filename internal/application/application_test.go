package application

import (
	"context"
	"testing"

	"github.com/jannfis/argocd-agent/internal/metrics"
	"github.com/jannfis/argocd-agent/server/backend/kubernetes"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
)

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
	exApp := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "existing",
			Namespace: "default",
		},
	}
	t.Run("Create a new application", func(t *testing.T) {
		m := NewManager(kubernetes.NewKubernetesBackend(fakeappclient.NewSimpleClientset(), nil))
		err := m.Create(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "default"}})
		assert.NoError(t, err)
	})

	t.Run("Create an application that exists", func(t *testing.T) {
		m := NewManager(kubernetes.NewKubernetesBackend(fakeappclient.NewSimpleClientset(exApp), nil))
		err := m.Create(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.Error(t, err)
	})
}

func Test_ManagerUpdateStatus(t *testing.T) {
	exApp := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "existing",
			Namespace: "default",
		},
	}
	t.Run("Update existing application", func(t *testing.T) {
		m := NewManager(kubernetes.NewKubernetesBackend(fakeappclient.NewSimpleClientset(exApp), nil))
		err := m.UpdateStatus(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.NoError(t, err)
	})

	t.Run("Update non-existing application", func(t *testing.T) {
		m := NewManager(kubernetes.NewKubernetesBackend(fakeappclient.NewSimpleClientset(), nil))
		err := m.UpdateStatus(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.Error(t, err)
	})

	t.Run("Upsert non-existing application", func(t *testing.T) {
		m := NewManager(kubernetes.NewKubernetesBackend(fakeappclient.NewSimpleClientset(), nil), WithAllowUpsert(true))
		err := m.UpdateStatus(context.TODO(), &v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "existing", Namespace: "default"}})
		assert.NoError(t, err)
	})

}
