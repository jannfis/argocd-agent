package agent

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakekube "github.com/jannfis/argocd-application-agent/test/fake/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(metric prometheus.Counter) float64 {
	var m = &dto.Metric{}
	if err := metric.Write(m); err != nil {
		return 0
	}
	return m.Counter.GetValue()
}

func Test_NewAgent(t *testing.T) {
	fakec := fakekube.NewFakeClientsetWithResources()
	appc := fakeappclient.NewSimpleClientset()
	agent := NewAgent(fakec, appc, WithAgentNamespace("test"))
	require.NotNil(t, agent)
	stopch := make(chan struct{})
	err := agent.Run(stopch)
	assert.NoError(t, err)
	app1 := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
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
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "bar",
			},
		},
	}
	app1, err = appc.ArgoprojV1alpha1().Applications("test").Create(context.TODO(), app1, v1.CreateOptions{})
	require.NoError(t, err)
	app1.Spec.Source.Path = "foobar"
	_, err = appc.ArgoprojV1alpha1().Applications("test").Update(context.TODO(), app2, v1.UpdateOptions{})
	require.NoError(t, err)
	stopch <- struct{}{}
	time.Sleep(500 * time.Millisecond)
}
