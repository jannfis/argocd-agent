package agent

import (
	"fmt"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// createApplication creates an Application upon an event in the agent's work
// queue.
func (a *Agent) createApplication(app *v1alpha1.Application) error {
	app.ObjectMeta.SetNamespace(a.namespace)
	logCtx := log().WithFields(logrus.Fields{
		"method": "CreateApplication",
		"app":    app.QualifiedName(),
	})

	// If we receive a new app event for an app we already manage, it usually
	// means that we're out-of-sync from the control plane.
	//
	// TODO(jannfis): Handle this situation properly instead of throwing an error.
	if a.appManager.IsManaged(app.QualifiedName()) {
		logCtx.Infof("App is already managed")
		return fmt.Errorf("application %s is already managed", app.QualifiedName())
	}

	logCtx.Infof("Creating a new application")

	// We start with an empty resource version and fresh generation
	app.ResourceVersion = ""
	app.Generation = 0
	if app.Annotations != nil {
		delete(app.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
	// We start with an empty status, regardless of what was provided
	app.Status = v1alpha1.ApplicationStatus{}

	// We set managed status early, so that the informer event for creation
	// of the app will be ignored. If the creation results in an error, we
	// mark the app as unmanaged.
	a.appManager.Manage(app.QualifiedName())
	_, err := a.appclient.ArgoprojV1alpha1().Applications(a.namespace).Create(a.context, app, v1.CreateOptions{})
	if err != nil {
		a.appManager.Unmanage(app.QualifiedName())
	}
	return err
}

func (a *Agent) updateApplication(app *v1alpha1.Application) error {
	app.ObjectMeta.SetNamespace(a.namespace)
	logCtx := log().WithFields(logrus.Fields{
		"method": "UpdateApplication",
		"app":    app.QualifiedName(),
	})

	// If we receive an update app event for an app we don't know about yet it
	// means that we're out-of-sync from the control plane.
	//
	// TODO(jannfis): Handle this situation properly instead of throwing an error.
	if !a.appManager.IsManaged(app.QualifiedName()) {
		return fmt.Errorf("application %s is not managed", app.QualifiedName())
	}

	logCtx.Infof("Updating application")

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		oapp, ierr := a.appclient.ArgoprojV1alpha1().Applications(a.namespace).Get(a.context, app.Name, v1.GetOptions{})
		if ierr != nil {
			return ierr
		}

		// We want annotations and labels from the updated app
		oapp.ObjectMeta.Annotations = app.ObjectMeta.Annotations
		oapp.ObjectMeta.Labels = app.ObjectMeta.Labels

		// Pull in any changes to the app's spec
		oapp.Spec = *app.Spec.DeepCopy()

		// Pull in any changes to the app's operation
		oapp.Operation = app.Operation.DeepCopy()

		a.watchLock.Lock()
		defer a.watchLock.Unlock()
		napp, ierr := a.appclient.ArgoprojV1alpha1().Applications(a.namespace).Update(a.context, oapp, v1.UpdateOptions{})
		if ierr == nil {
			a.appManager.IgnoreChange(napp.QualifiedName(), napp.ResourceVersion)
		}
		return ierr

	})
	if err != nil {
		logCtx.WithError(err).Error("Could not update application")
		a.appManager.Unmanage(app.QualifiedName())
	}
	return err
}

func (a *Agent) deleteApplication(app *v1alpha1.Application) error {
	app.ObjectMeta.SetNamespace(a.namespace)
	logCtx := log().WithFields(logrus.Fields{
		"method": "DeleteApplication",
		"app":    app.QualifiedName(),
	})

	// If we receive an update app event for an app we don't know about yet it
	// means that we're out-of-sync from the control plane.
	//
	// TODO(jannfis): Handle this situation properly instead of throwing an error.
	if !a.appManager.IsManaged(app.QualifiedName()) {
		return fmt.Errorf("application %s is not managed", app.QualifiedName())
	}

	logCtx.Infof("Deleting application")

	delPol := v1.DeletePropagationBackground
	err := a.appclient.ArgoprojV1alpha1().Applications(a.namespace).Delete(a.context, app.Name, v1.DeleteOptions{PropagationPolicy: &delPol})
	if err != nil {
		return err
	}
	a.appManager.Unmanage(app.QualifiedName())
	return nil
}

func (a *Agent) copyAndMutateApp(app *v1alpha1.Application, create bool) *v1alpha1.Application {
	rapp := &v1alpha1.Application{}
	rapp.ObjectMeta.Annotations = app.ObjectMeta.Annotations
	rapp.ObjectMeta.Labels = app.ObjectMeta.Labels
	rapp.ObjectMeta.Namespace = a.namespace
	rapp.Spec = *app.Spec.DeepCopy()
	rapp.Operation = app.Operation.DeepCopy()
	if rapp.ObjectMeta.Annotations != nil {
		delete(rapp.ObjectMeta.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
	return rapp
}
