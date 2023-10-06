package appinformer

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Test_AppInformer(t *testing.T) {
	app1 := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "foo",
			},
		},
	}
	app2 := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test2",
			Namespace: "test",
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "bar",
			},
		},
	}
	t.Run("Simple list callback", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1, app2)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "test", WithListAppCallback(func(apps []v1alpha1.Application) []v1alpha1.Application {
			eventCh <- true
			return apps
		}))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(1 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		apps, err := inf.Lister.Applications(inf.options.namespace).List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 2)
	})

	t.Run("List callback with filter", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1, app2)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "test", WithListAppCallback(func(apps []v1alpha1.Application) []v1alpha1.Application {
			eventCh <- true
			return []v1alpha1.Application{*app1}
		}))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(1 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		apps, err := inf.Lister.Applications(inf.options.namespace).List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		app, err := inf.Lister.Applications(inf.options.namespace).Get("test1")
		assert.NoError(t, err)
		assert.NotNil(t, app)
		app, err = inf.Lister.Applications(inf.options.namespace).Get("test2")
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, app)
	})

	t.Run("Add callback", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset()
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "test", WithNewAppCallback(func(app *v1alpha1.Application) {
			eventCh <- true
		}))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(1 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				fac.ArgoprojV1alpha1().Applications(inf.options.namespace).Create(context.TODO(), app1, v1.CreateOptions{})
				time.Sleep(100 * time.Millisecond)
			}
		}
		apps, err := inf.Lister.Applications(inf.options.namespace).List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		app, err := inf.Lister.Applications(inf.options.namespace).Get("test1")
		assert.NoError(t, err)
		assert.NotNil(t, app)
		app, err = inf.Lister.Applications(inf.options.namespace).Get("test2")
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, app)
	})

	t.Run("Update callback", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "test", WithUpdateAppCallback(func(old *v1alpha1.Application, new *v1alpha1.Application) {
			eventCh <- true
		}))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(2 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
				appc := app1.DeepCopy()
				appc.Spec.Project = "hello"
				fac.ArgoprojV1alpha1().Applications(inf.options.namespace).Update(context.TODO(), appc, v1.UpdateOptions{})
			}
		}
		apps, err := inf.Lister.Applications(inf.options.namespace).List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		napp, err := inf.Lister.Applications(inf.options.namespace).Get("test1")
		assert.NoError(t, err)
		assert.NotNil(t, napp)
		assert.Equal(t, "hello", napp.Spec.Project)
	})

	t.Run("Delete callback", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "test", WithDeleteAppCallback(func(app *v1alpha1.Application) {
			eventCh <- true
		}))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(2 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
				fac.ArgoprojV1alpha1().Applications(inf.options.namespace).Delete(context.TODO(), "test1", v1.DeleteOptions{})
			}
		}
		apps, err := inf.Lister.Applications(inf.options.namespace).List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 0)
		napp, err := inf.Lister.Applications(inf.options.namespace).Get("test1")
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, napp)
	})

	t.Run("Test admission in forbidden namespace", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "default", WithListAppCallback(func(apps []v1alpha1.Application) []v1alpha1.Application {
			eventCh <- true
			return apps
		}), WithNamespaces("kube-system"))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(2 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		apps, err := inf.Lister.Applications("").List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 0)
		napp, err := inf.Lister.Applications("").Get("test1")
		assert.ErrorContains(t, err, "not found")
		assert.Nil(t, napp)
	})

	t.Run("Test admission in allowed namespace", func(t *testing.T) {
		fac := fakeappclient.NewSimpleClientset(app1)
		eventCh := make(chan interface{})
		inf := NewAppInformer(fac, "default", WithListAppCallback(func(apps []v1alpha1.Application) []v1alpha1.Application {
			eventCh <- true
			return apps
		}), WithNamespaces("test"))
		stopCh := make(chan struct{})
		go func() {
			inf.Informer.Run(stopCh)
		}()
		ticker := time.NewTicker(2 * time.Second)
		running := true
		for running {
			select {
			case <-ticker.C:
				t.Fatal("callback timeout reached")
			case <-eventCh:
				time.Sleep(500 * time.Millisecond)
				running = false
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		apps, err := inf.Lister.Applications("").List(labels.Everything())
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		napp, err := inf.Lister.Applications("test").Get("test1")
		assert.NoError(t, err)
		assert.NotNil(t, napp)
	})
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
