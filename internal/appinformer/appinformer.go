package appinformer

import (
	"context"
	"strings"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	applisters "github.com/argoproj/argo-cd/v2/pkg/client/listers/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/glob"
	"github.com/jannfis/argocd-application-agent/internal/filter"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/sirupsen/logrus"
)

const defaultResyncPeriod = 1 * time.Hour

// AppInformer is a filtering and customizable SharedIndexInformer for Argo CD
// Application resources in a cluster. It works across a configurable set of
// namespaces, lets you define a list of filters to indicate interest in the
// resource of a particular even and allows you to set up callbacks to handle
// the events.
type AppInformer struct {
	appClient appclientset.Interface

	options *AppInformerOptions

	Informer cache.SharedIndexInformer
	Lister   applisters.ApplicationLister
}

// ListAppsCallback is executed when the informer builds its cache. It receives
// all apps matching the configured label selector and must returned a list of
// apps to keep in the cache.
//
// Callbacks are executed after the AppInformer validated that the application
// is good for processing.
type ListAppsCallback func(apps []v1alpha1.Application) []v1alpha1.Application

// NewAppCallback is executed when a new app is determined by the underlying
// watcher.
//
// Callbacks are executed after the AppInformer validated that the application
// is allowed for processing.
type NewAppCallback func(app *v1alpha1.Application)

// UpdateAppCallback is executed when a change event for an app is determined
// by the underlying watcher.
//
// Callbacks are executed after the AppInformer validated that the application
// is allowed for processing.
type UpdateAppCallback func(old *v1alpha1.Application, new *v1alpha1.Application)

// DeleteAppCallback is executed when an app delete event is determined by the
// underlying watcher.
//
// Callbacks are executed after the AppInformer validated that the application
// is allowed for processing.
type DeleteAppCallback func(app *v1alpha1.Application)

// ErrorCallback is executed when the watcher events encounter an error
type ErrorCallback func(err error, fmt string, args ...string)

// NewAppInformer returns a new application informer for a given namespace
func NewAppInformer(client appclientset.Interface, namespace string, opts ...AppInformerOption) *AppInformer {
	o := &AppInformerOptions{
		resync: defaultResyncPeriod,
	}
	o.filters = filter.NewFilterChain()
	for _, opt := range opts {
		opt(o)
	}
	if len(o.namespaces) > 0 {
		o.namespace = ""
	} else {
		o.namespace = namespace
	}

	i := &AppInformer{options: o, appClient: client}

	i.Informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				logCtx := log().WithField("component", "ListWatch")
				logCtx.Debugf("Listing apps into cache")
				appList, err := i.appClient.ArgoprojV1alpha1().Applications(o.namespace).List(context.TODO(), options)
				if err != nil {
					logCtx.Warnf("Error listing apps: %v", err)
					return nil, err
				}

				// The result of the list call will get pre-filtered to only
				// contain apps that a) are in a namespace we are allowed to
				// process and b) pass admission through the informer's chain
				// of configured filters.
				preFilteredItems := make([]v1alpha1.Application, 0)
				for _, app := range appList.Items {
					if i.isAppAllowed(&app) {
						preFilteredItems = append(preFilteredItems, app)
						logCtx.Tracef("Allowing app %s in namespace %s", app.Name, app.Namespace)
					} else {
						logCtx.Tracef("Not allowing app %s in namespace %s", app.Name, app.Namespace)
					}
				}

				// The pre-filtered list is passed to the configured callback
				// to perform custom filtering and tranformation.
				if i.options.listCb != nil {
					newItems := i.options.listCb(preFilteredItems)
					appList.Items = newItems
				}

				return appList, err
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				logCtx := log().WithField("component", "WatchFunc")
				logCtx.Info("Starting application watcher")
				return i.appClient.ArgoprojV1alpha1().Applications(namespace).Watch(context.TODO(), options)
			},
		},
		&v1alpha1.Application{},
		i.options.resync,
		cache.Indexers{
			cache.NamespaceIndex: func(obj interface{}) ([]string, error) {
				return cache.MetaNamespaceIndexFunc(obj)
			},
		},
	)
	i.Informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				logCtx := log().WithField("component", "AddFunc")
				logCtx.Tracef("New application event")
				app, ok := obj.(*v1alpha1.Application)
				if !ok || app == nil {
					// if i.options.errorCb != nil {
					// 	i.options.errorCb(nil, "invalid resource received by add event")
					// }
					return
				}
				if !i.isAppAllowed(app) {
					return
				}
				if i.options.newCb != nil {
					i.options.newCb(app)
				}
				if i.options.appMetrics != nil {
					i.options.appMetrics.AddApp()
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				logCtx := log().WithField("component", "UpdateFunc")
				logCtx.Tracef("Application update event")
				newApp, newOk := newObj.(*v1alpha1.Application)
				oldApp, oldOk := oldObj.(*v1alpha1.Application)
				if !newOk || !oldOk {
					// if i.options.errorCb != nil {
					// 	i.options.errorCb(nil, "invalid resource received by update event")
					// }
					return
				}
				logCtx = logCtx.WithField("application", newApp.Name)

				// Namespace of new and old app must match. Theoretically, they
				// should always match, but we safeguard.
				if oldApp.Namespace != newApp.Namespace {
					logCtx.Warnf("namespace mismatch between old and new app")
					return
				}

				if !i.isAppAllowed(newApp) {
					logCtx.Tracef("application not allowed")
					return
				}

				if !i.options.filters.ProcessChange(oldApp, newApp) {
					logCtx.Debugf("Change will not be processed")
					return
				}
				if i.options.updateCb != nil {
					i.options.updateCb(oldApp, newApp)
				}
				if i.options.appMetrics != nil {
					i.options.appMetrics.UpdateApp()
				}
			},
			DeleteFunc: func(obj interface{}) {
				logCtx := log().WithField("component", "DeleteFunc")
				logCtx.Tracef("Application update event")
				app, ok := obj.(*v1alpha1.Application)
				if !ok || app == nil {
					// if i.options.errorCb != nil {
					// 	i.options.errorCb(nil, "invalid resource received by delete event")
					// }
					return
				}
				logCtx = logCtx.WithField("application", app.QualifiedName())
				if !i.isAppAllowed(app) {
					logCtx.Tracef("Ignoring application delete event")
					return
				}
				if i.options.deleteCb != nil {
					i.options.deleteCb(app)
				}
				if i.options.appMetrics != nil {
					i.options.appMetrics.RemoveApp()
				}
			},
		},
	)
	i.Lister = applisters.NewApplicationLister(i.Informer.GetIndexer())
	i.Informer.SetWatchErrorHandler(cache.DefaultWatchErrorHandler)
	return i
}

func (i *AppInformer) Start(stopch <-chan struct{}) {
	log().Infof("Starting app informer (namespaces: %s)", strings.Join(append([]string{i.options.namespace}, i.options.namespaces...), ","))
	i.Informer.Run(stopch)
}

// isAppAllowed returns true if the app is allowed to be processed
func (i *AppInformer) isAppAllowed(app *v1alpha1.Application) bool {
	return glob.MatchStringInList(append([]string{i.options.namespace}, i.options.namespaces...), app.Namespace, false) &&
		i.options.filters.Admit(app)
}

func log() *logrus.Entry {
	return logrus.WithField("module", "informer")
}

func init() {
}
