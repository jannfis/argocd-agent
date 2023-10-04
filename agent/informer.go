package agent

import (
	"context"
	"reflect"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	log "github.com/sirupsen/logrus"
)

// newInformer configures and returns a new shared index informer to handle
// all Application resources we're about to manage.
func (a *Agent) newInformer() cache.SharedIndexInformer {
	namespace := a.opts.namespace
	if len(a.opts.namespaces) > 0 {
		namespace = ""
	}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				logCtx := log.WithField("component", "informer")
				logCtx.Infof("Listing applications")
				appList, err := a.appclient.ArgoprojV1alpha1().Applications(namespace).List(context.TODO(), options)
				if err != nil {
					logCtx.WithError(err).Debugf("Error listing applications")
					return nil, err
				}
				newItems := make([]v1alpha1.Application, 0)
				for _, app := range appList.Items {
					// Drop any Application that's not admitted from the cache
					if a.filters.Admit(&app) {
						a.appQueue.Add(&app)
						newItems = append(newItems, app)
					}
				}
				appList.Items = newItems
				return appList, err
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				logCtx := log.WithField("component", "watcher")
				logCtx.Info("Starting watcher")
				return a.appclient.ArgoprojV1alpha1().Applications(namespace).Watch(context.TODO(), options)
			},
		},
		&v1alpha1.Application{},
		1*time.Hour,
		cache.Indexers{
			cache.NamespaceIndex: func(obj interface{}) ([]string, error) {
				return cache.MetaNamespaceIndexFunc(obj)
			},
		},
	)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				logCtx := log.
					WithField("component", "watcher").
					WithField("event", "add application")

				app, ok := obj.(*v1alpha1.Application)
				if !ok || app == nil {
					logCtx.Warnf("Received invalid resource")
					a.metrics.app.Error()
					return
				}

				logCtx = logCtx.
					WithField("application", app.GetName()).
					WithField("namespace", app.GetNamespace())

				if a.filters.Admit(app) {
					logCtx.Infof("New application")
					a.metrics.app.AddApp()
					a.appQueue.Add(app)
				} else {
					logCtx.Trace("Ignoring event because admission filter returned false")
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				newApp, newOk := newObj.(*v1alpha1.Application)
				oldApp, oldOk := oldObj.(*v1alpha1.Application)
				if !newOk || !oldOk {
					a.metrics.app.Error()
					log.Warnf("informer received malformed update event")
					return
				}
				// Return early if we're not interested in this change
				if !a.filters.ProcessChange(oldApp, newApp) {
					return
				}
				log.
					WithField("application", newApp.GetName()).
					WithField("namespace", newApp.GetNamespace()).
					WithField("status_changed", !reflect.DeepEqual(newApp.Status, oldApp.Status)).
					WithField("spec_changed", !reflect.DeepEqual(newApp.Spec, oldApp.Spec)).
					Infof("Status changed")

				a.metrics.app.UpdateApp()
				a.appQueue.Add(newApp)
			},
			DeleteFunc: func(obj interface{}) {
				a.metrics.app.RemoveApp()
			},
		},
	)
	informer.SetWatchErrorHandler(cache.DefaultWatchErrorHandler)
	return informer
}
