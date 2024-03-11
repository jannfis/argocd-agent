package appproject

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AppProjectInformer(t *testing.T) {
	numAdded := atomic.Uint32{}
	numUpdated := atomic.Uint32{}
	numDeleted := atomic.Uint32{}
	ac := fakeappclient.NewSimpleClientset()
	pi, lister, err := NewAppProjectInformer(context.TODO(), ac,
		WithAddFunc(func(proj *v1alpha1.AppProject) {
			numAdded.Add(1)
		}),
		WithUpdateFunc(func(oldProj, newProj *v1alpha1.AppProject) {
			numUpdated.Add(1)
			require.NotEqual(t, oldProj.Spec.Description, newProj.Spec.Description)
		}),
		WithDeleteFunc(func(proj *v1alpha1.AppProject) {
			numDeleted.Add(1)
		}),
		WithNamespaces("argocd"),
	)
	require.NoError(t, err)
	require.NotNil(t, pi)
	err = pi.Start(context.TODO())
	require.NoError(t, err)
	for !pi.IsSynced() {
		time.Sleep(100 * time.Millisecond)
	}
	t.Run("Add AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			ac.ArgoprojV1alpha1().AppProjects("argocd").Create(context.TODO(), &v1alpha1.AppProject{
				ObjectMeta: v1.ObjectMeta{Name: fmt.Sprintf("proj%d", i)},
			}, v1.CreateOptions{})
		}
		for numAdded.Load() != 5 {
			time.Sleep(100 * time.Millisecond)
		}
		assert.EqualValues(t, 5, numAdded.Load())
		// Projects should be available from the indexer now
		for _, i := range []int{1, 2, 3, 4, 5} {
			n := fmt.Sprintf("proj%d", i)
			p, err := lister.AppProjects("argocd").Get(n)
			require.NotNil(t, p)
			require.NoError(t, err)
			assert.Equal(t, n, p.Name)
		}
	})
	t.Run("Modify AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			p, err := ac.ArgoprojV1alpha1().AppProjects("argocd").Get(context.TODO(), fmt.Sprintf("proj%d", i), v1.GetOptions{})
			require.NoError(t, err)
			p.Spec.Description = "foobar"
			ac.ArgoprojV1alpha1().AppProjects("argocd").Update(context.TODO(), p, v1.UpdateOptions{})
		}
		for numUpdated.Load() != 5 {
			time.Sleep(100 * time.Millisecond)
		}
		assert.EqualValues(t, 5, numUpdated.Load())
	})

	t.Run("Delete AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			err := ac.ArgoprojV1alpha1().AppProjects("argocd").Delete(context.TODO(), fmt.Sprintf("proj%d", i), v1.DeleteOptions{})
			require.NoError(t, err)
		}
		for numDeleted.Load() != 5 {
			time.Sleep(100 * time.Millisecond)
		}
		assert.EqualValues(t, 5, numDeleted.Load())
	})
}

func Test_FilterFunc(t *testing.T) {
	numAdded := atomic.Uint32{}
	numUpdated := atomic.Uint32{}
	numDeleted := atomic.Uint32{}
	ac := fakeappclient.NewSimpleClientset()
	pi, lister, err := NewAppProjectInformer(context.TODO(), ac,
		WithAddFunc(func(proj *v1alpha1.AppProject) {
			numAdded.Add(1)
			if numAdded.Load() > 1 {
				t.Fatalf("AddFunc called for %s", proj.GetName())
			}
		}),
		WithUpdateFunc(func(oldProj, newProj *v1alpha1.AppProject) {
			numUpdated.Add(1)
			if numUpdated.Load() > 1 {
				t.Fatalf("UpdateFunc called for %s", newProj.GetName())
			}
		}),
		WithDeleteFunc(func(proj *v1alpha1.AppProject) {
			numDeleted.Add(1)
			if numDeleted.Load() > 1 {
				t.Fatalf("DeleteFunc called for %s", proj.GetName())
			}
		}),
		WithListFilter(func(proj *v1alpha1.AppProject) bool {
			if proj.Name == "proj1" {
				return true
			} else {
				return false
			}
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, pi)
	err = pi.Start(context.TODO())
	require.NoError(t, err)
	for !pi.IsSynced() {
		time.Sleep(100 * time.Millisecond)
	}

	t.Run("Add AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			ac.ArgoprojV1alpha1().AppProjects("argocd").Create(context.TODO(), &v1alpha1.AppProject{
				ObjectMeta: v1.ObjectMeta{Name: fmt.Sprintf("proj%d", i)},
			}, v1.CreateOptions{})
		}
		p, err := lister.AppProjects("argocd").Get("proj1")
		assert.NotNil(t, p)
		assert.NoError(t, err)
	})
	t.Run("Update AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			ac.ArgoprojV1alpha1().AppProjects("argocd").Update(context.TODO(), &v1alpha1.AppProject{
				ObjectMeta: v1.ObjectMeta{Name: fmt.Sprintf("proj%d", i)}, Spec: v1alpha1.AppProjectSpec{Description: "Foo"},
			}, v1.UpdateOptions{})
		}
	})
	t.Run("Delete AppProjects", func(t *testing.T) {
		for _, i := range []int{1, 2, 3, 4, 5} {
			ac.ArgoprojV1alpha1().AppProjects("argocd").Delete(context.TODO(), fmt.Sprintf("proj%d", i), v1.DeleteOptions{})
		}
	})

}

func Test_NamespaceNotAllowed(t *testing.T) {

}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
