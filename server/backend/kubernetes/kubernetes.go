/*
Package kubernetes implements an Application backend that uses a Kubernetes
informer to keep track of resources, and an appclientset to manipulate
Application resources on the cluster.
*/
package kubernetes

import (
	"context"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	"github.com/jannfis/argocd-agent/internal/appinformer"
	"github.com/jannfis/argocd-agent/server/backend"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ backend.Application = &KubernetesBackend{}

type KubernetesBackend struct {
	appClient appclientset.Interface
	informer  *appinformer.AppInformer
	namespace string
}

func NewKubernetesBackend(appClient appclientset.Interface, informer *appinformer.AppInformer) *KubernetesBackend {
	return &KubernetesBackend{
		appClient: appClient,
		informer:  informer,
	}
}

func (be *KubernetesBackend) List(ctx context.Context, selector backend.ApplicationSelector) ([]v1alpha1.Application, error) {
	res := make([]v1alpha1.Application, 0)
	if len(selector.Namespaces) > 0 {
		for _, ns := range selector.Namespaces {
			l, err := be.appClient.ArgoprojV1alpha1().Applications(ns).List(ctx, v1.ListOptions{})
			if err != nil {
				return nil, err
			}
			res = append(res, l.Items...)
		}
	} else {
		l, err := be.appClient.ArgoprojV1alpha1().Applications("").List(ctx, v1.ListOptions{})
		if err != nil {
			return nil, err
		}
		res = append(res, l.Items...)
	}
	return res, nil
}

func (be *KubernetesBackend) Create(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	return be.appClient.ArgoprojV1alpha1().Applications(app.Namespace).Create(ctx, app, v1.CreateOptions{})
}

func (be *KubernetesBackend) Get(ctx context.Context, name string, namespace string) (*v1alpha1.Application, error) {
	return be.appClient.ArgoprojV1alpha1().Applications(namespace).Get(ctx, name, v1.GetOptions{})
}

func (be *KubernetesBackend) Delete(ctx context.Context, name string, namespace string) error {
	return be.appClient.ArgoprojV1alpha1().Applications(namespace).Delete(ctx, name, v1.DeleteOptions{})
}

func (be *KubernetesBackend) Update(ctx context.Context, app *v1alpha1.Application) (*v1alpha1.Application, error) {
	return be.appClient.ArgoprojV1alpha1().Applications(app.Namespace).Update(ctx, app, v1.UpdateOptions{})
}

func (be *KubernetesBackend) StartInformer(ctx context.Context) {
	be.informer.Start(ctx.Done())
}
