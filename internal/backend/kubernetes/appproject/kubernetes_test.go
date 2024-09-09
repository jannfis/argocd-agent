// Copyright 2024 The argocd-agent Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/argoproj-labs/argocd-agent/internal/backend"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wI2L/jsondiff"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_NewKubernetes(t *testing.T) {
	t.Run("New with patch capability", func(t *testing.T) {
		k := NewKubernetesBackend(nil, "", nil, true)
		assert.NotNil(t, k)
		assert.True(t, k.SupportsPatch())
	})
	t.Run("New without patch capability", func(t *testing.T) {
		k := NewKubernetesBackend(nil, "", nil, false)
		assert.NotNil(t, k)
		assert.False(t, k.SupportsPatch())
	})

}

func mkAppProjects() []runtime.Object {
	appProjects := make([]runtime.Object, 10)
	for i := 0; i < 10; i += 1 {
		appProjects[i] = runtime.Object(&v1alpha1.AppProject{
			ObjectMeta: v1.ObjectMeta{
				Name:      "app-project" + fmt.Sprintf("%d", i),
				Namespace: fmt.Sprintf("ns%d", i),
			},
		})
	}
	return appProjects
}

func Test_List(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("No appProjects", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset()
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		appProjects, err := k.List(context.TODO(), backend.AppProjectSelector{})
		require.NoError(t, err)
		assert.Len(t, appProjects, 0)
	})
	t.Run("Few appProjects", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		appProjects, err := k.List(context.TODO(), backend.AppProjectSelector{})
		require.NoError(t, err)
		assert.Len(t, appProjects, 10)
	})
	t.Run("appProjects with matching selector", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		appProjects, err := k.List(context.TODO(), backend.AppProjectSelector{Names: []string{"app-project1", "app-project2"}})
		require.NoError(t, err)
		assert.Len(t, appProjects, 2)
	})

	t.Run("appProjects with non-matching selector", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		appProjects, err := k.List(context.TODO(), backend.AppProjectSelector{Names: []string{"app-project1", "app-project2"}})
		require.NoError(t, err)
		assert.Len(t, appProjects, 0)
	})
}

func Test_Create(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("Create app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Create(context.TODO(), &v1alpha1.AppProject{ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "bar"}})
		assert.NoError(t, err)
		assert.NotNil(t, app)
	})
	t.Run("Create existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Create(context.TODO(), &v1alpha1.AppProject{ObjectMeta: v1.ObjectMeta{Name: "app", Namespace: "ns1"}})
		assert.ErrorContains(t, err, "exists")
		assert.Nil(t, app)
	})
}

func Test_Get(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("Get existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Get(context.TODO(), "app", "ns1")
		assert.NoError(t, err)
		assert.NotNil(t, app)
	})
	t.Run("Get non-existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Get(context.TODO(), "foo", "ns1")
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, app)
	})
}

func Test_Delete(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("Delete existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		deletionPropagation := backend.DeletePropagationForeground
		err := k.Delete(context.TODO(), "app", "ns1", &deletionPropagation)
		assert.NoError(t, err)
	})
	t.Run("Delete non-existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		deletionPropagation := backend.DeletePropagationForeground
		err := k.Delete(context.TODO(), "app", "ns10", &deletionPropagation)
		assert.ErrorContains(t, err, "not found")
	})
}

func Test_Update(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("Update existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Update(context.TODO(), &v1alpha1.AppProject{ObjectMeta: v1.ObjectMeta{Name: "app", Namespace: "ns1"}})
		assert.NoError(t, err)
		assert.NotNil(t, app)
	})
	t.Run("Update non-existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Update(context.TODO(), &v1alpha1.AppProject{ObjectMeta: v1.ObjectMeta{Name: "app", Namespace: "ns10"}})
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, app)
	})
}

func Test_Patch(t *testing.T) {
	appProjects := mkAppProjects()
	t.Run("Patch existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		p := jsondiff.Patch{jsondiff.Operation{Type: "add", Path: "/foo", Value: "bar"}}
		jsonpatch, err := json.Marshal(p)
		require.NoError(t, err)
		app, err := k.Patch(context.TODO(), "app", "ns1", jsonpatch)
		assert.NoError(t, err)
		assert.NotNil(t, app)
	})
	t.Run("Update non-existing app", func(t *testing.T) {
		fakeAppC := fakeappclient.NewSimpleClientset(appProjects...)
		k := NewKubernetesBackend(fakeAppC, "", nil, true)
		app, err := k.Update(context.TODO(), &v1alpha1.AppProject{ObjectMeta: v1.ObjectMeta{Name: "app", Namespace: "ns10"}})
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, app)
	})
}