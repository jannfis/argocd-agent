package agent

import (
	"github.com/jannfis/argocd-application-agent/internal/appinformer"
	"github.com/jannfis/argocd-application-agent/internal/batch"
	"github.com/jannfis/argocd-application-agent/internal/filter"

	// "github.com/jannfis/argocd-application-agent/internal/filter"
	"github.com/jannfis/argocd-application-agent/internal/metrics"
	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	applisters "github.com/argoproj/argo-cd/v2/pkg/client/listers/application/v1alpha1"
)

type agentMetrics struct {
	app *metrics.ApplicationMetrics
}

// Agent is a controller that synchronizes Application resources
type Agent struct {
	client    kubernetes.Interface
	appclient appclientset.Interface
	opts      AgentOptions

	appInformer cache.SharedIndexInformer
	appLister   applisters.ApplicationLister

	informer *appinformer.AppInformer

	appQueue *batch.AutoBatchQueue

	metrics agentMetrics

	filters *filter.Chain
}

// AgentOptions defines the options for a given Controller
type AgentOptions struct {
	namespace  string
	namespaces []string
}

type AgentOption func(*AgentOptions)

func (a *Agent) informerListCallback(apps []v1alpha1.Application) []v1alpha1.Application {
	newApps := make([]v1alpha1.Application, 0)
	for _, app := range apps {
		if a.filters.Admit(&app) {
			newApps = append(newApps, app)
		}
	}
	return newApps
}

// NewAgent creates a new agent instance, using the given client interfaces and
// options.
func NewAgent(client kubernetes.Interface, appclient appclientset.Interface, opts ...AgentOption) *Agent {
	a := &Agent{}
	a.client = client
	a.appclient = appclient
	for _, o := range opts {
		o(&a.opts)
	}

	// a.appInformer = a.newInformer()
	// a.appLister = applisters.NewApplicationLister(a.appInformer.GetIndexer())

	// Set up metrics providers
	a.metrics.app = metrics.NewApplicationMetrics()

	cbFunc := batch.WithCallbackFunc(func(bq *batch.AutoBatchQueue) {
		log.Infof("Batch!")
	})

	a.appQueue = batch.NewAutoBatch(batch.WithBatchSize(1), cbFunc)

	// Set up default filter chain
	// a.filters = a.DefaultFilterChain()

	a.informer = appinformer.NewAppInformer(a.appclient, a.opts.namespace, appinformer.WithMetrics(a.metrics.app))
	return a
}

func (a *Agent) Run(stopchan chan struct{}) error {
	log.Infof("Starting Argo CD agent (ns=%s, allowed_namespaces=%v)", a.opts.namespace, a.opts.namespaces)
	go func() {
		a.informer.Informer.Run(stopchan)
	}()
	return nil
}

func (a *Agent) Stop() error {
	return nil
}
