package agent

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	fakeappclient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jannfis/argocd-agent/internal/event"
	fakekube "github.com/jannfis/argocd-agent/test/fake/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewAgent(t *testing.T) {
	fakec := fakekube.NewFakeClientsetWithResources()
	appc := fakeappclient.NewSimpleClientset()
	agent, err := NewAgent(context.TODO(), fakec, appc, "agent")
	require.NotNil(t, agent)
	require.NoError(t, err)
}

func Test_AgentNewAppFromInformer(t *testing.T) {
	fakec := fakekube.NewFakeClientsetWithResources()
	appc := fakeappclient.NewSimpleClientset()
	agent, err := NewAgent(context.TODO(), fakec, appc, "agent")
	require.NotNil(t, agent)
	require.NoError(t, err)
	err = agent.Start(context.Background())
	require.NoError(t, err)

	t.Run("Add application event when agent is not connected", func(t *testing.T) {
		syncch := make(chan bool)
		ocb := agent.informer.NewAppCallback()
		ncb := func(app *v1alpha1.Application) {
			ocb(app)
			syncch <- true
		}
		agent.informer.SetNewAppCallback(ncb)
		defer agent.informer.SetNewAppCallback(ocb)
		agent.informer.EnsureSynced(1 * time.Second)
		_, err := appc.ArgoprojV1alpha1().Applications(agent.namespace).Create(agent.context, &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "testapp1",
				Namespace: agent.namespace,
			},
		}, v1.CreateOptions{})
		require.NoError(t, err)
		// Wait for the informer callback to finish
		<-syncch
		assert.Equal(t, 0, agent.queues.SendQ(defaultQueueName).Len())
	})

	t.Run("Add application event when agent is connected", func(t *testing.T) {
		syncch := make(chan bool)
		ocb := agent.informer.NewAppCallback()
		ncb := func(app *v1alpha1.Application) {
			ocb(app)
			syncch <- true
		}
		agent.informer.SetNewAppCallback(ncb)
		defer agent.informer.SetNewAppCallback(ocb)
		agent.connected.Store(true)
		agent.informer.EnsureSynced(1 * time.Second)
		_, err := appc.ArgoprojV1alpha1().Applications(agent.namespace).Create(agent.context, &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "testapp2",
				Namespace: agent.namespace,
			},
		}, v1.CreateOptions{})
		require.NoError(t, err)
		// Wait for the informer callback to finish
		<-syncch
		assert.Equal(t, 1, agent.queues.SendQ(defaultQueueName).Len())
		ev, _ := agent.queues.SendQ(defaultQueueName).Get()
		require.IsType(t, event.Event{}, ev)
		assert.Equal(t, event.EventTypeAddApp, ev.(event.Event).Type)
		assert.Equal(t, "testapp2", ev.(event.Event).Application.Name)
	})

	time.Sleep(1 * time.Second)
	err = agent.Stop()
	require.NoError(t, err)
}

func Test_AgentUpdateAppFromInformer(t *testing.T) {
	app := &v1alpha1.Application{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testapp",
			Namespace: "agent",
		},
	}
	fakec := fakekube.NewFakeClientsetWithResources()
	appc := fakeappclient.NewSimpleClientset(app)
	agent, err := NewAgent(context.TODO(), fakec, appc, "agent")
	require.NotNil(t, agent)
	require.NoError(t, err)
	err = agent.Start(context.Background())
	require.NoError(t, err)

	t.Run("Update application event when agent is connected", func(t *testing.T) {
		syncch := make(chan bool)
		ocb := agent.informer.UpdateAppCallback()
		ncb := func(old *v1alpha1.Application, new *v1alpha1.Application) {
			ocb(old, new)
			syncch <- true
		}
		agent.informer.SetUpdateAppCallback(ncb)
		defer agent.informer.SetUpdateAppCallback(ocb)
		agent.connected.Store(true)
		agent.informer.EnsureSynced(1 * time.Second)
		_, err := appc.ArgoprojV1alpha1().Applications(agent.namespace).Update(agent.context, app, v1.UpdateOptions{})
		require.NoError(t, err)
		<-syncch
		// assert.Equal(t, 1, agent.queues.SendQ(defaultQueueName).Len())
		ev, _ := agent.queues.SendQ(defaultQueueName).Get()
		require.IsType(t, event.Event{}, ev)
		assert.Equal(t, event.EventTypeUpdateAppStatus, ev.(event.Event).Type)
		assert.Equal(t, "testapp", ev.(event.Event).Application.Name)
	})

}

func Test_AgentDeleteAppFromInformer(t *testing.T) {
	t.Run("Delete application event when agent is not connected", func(t *testing.T) {
		app := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "testapp",
				Namespace: "agent",
			},
		}
		fakec := fakekube.NewFakeClientsetWithResources()
		appc := fakeappclient.NewSimpleClientset(app)
		agent, err := NewAgent(context.TODO(), fakec, appc, "agent")
		require.NotNil(t, agent)
		require.NoError(t, err)
		err = agent.Start(context.Background())
		require.NoError(t, err)

		syncch := make(chan bool)
		ocb := agent.informer.DeleteAppCallback()
		ncb := func(app *v1alpha1.Application) {
			ocb(app)
			syncch <- true
		}
		agent.informer.SetDeleteAppCallback(ncb)
		defer agent.informer.SetDeleteAppCallback(ocb)
		agent.informer.EnsureSynced(1 * time.Second)
		err = appc.ArgoprojV1alpha1().Applications(agent.namespace).Delete(agent.context, app.Name, v1.DeleteOptions{})
		require.NoError(t, err)
		// Wait for the informer callback to finish
		<-syncch
		assert.Equal(t, 0, agent.queues.SendQ(defaultQueueName).Len())
	})

	t.Run("Delete application event when agent is connected", func(t *testing.T) {
		app := &v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "testapp",
				Namespace: "agent",
			},
		}
		fakec := fakekube.NewFakeClientsetWithResources()
		appc := fakeappclient.NewSimpleClientset(app)
		agent, err := NewAgent(context.TODO(), fakec, appc, "agent")
		require.NotNil(t, agent)
		require.NoError(t, err)
		err = agent.Start(context.Background())
		require.NoError(t, err)

		syncch := make(chan bool)
		ocb := agent.informer.DeleteAppCallback()
		ncb := func(app *v1alpha1.Application) {
			ocb(app)
			syncch <- true
		}
		agent.informer.SetDeleteAppCallback(ncb)
		defer agent.informer.SetDeleteAppCallback(ocb)
		agent.connected.Store(true)
		agent.informer.EnsureSynced(1 * time.Second)
		err = appc.ArgoprojV1alpha1().Applications(agent.namespace).Delete(agent.context, app.Name, v1.DeleteOptions{})
		require.NoError(t, err)
		// Wait for the informer callback to finish
		<-syncch
		assert.Equal(t, 1, agent.queues.SendQ(defaultQueueName).Len())
	})

}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
