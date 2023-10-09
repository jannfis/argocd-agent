package server

import "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"

// newAppCallback is a callback for the new app event
func (s *Server) newAppCallback(app *v1alpha1.Application) {
	logCtx := log().WithField("component", "NewAppCallback")
	// if !s.queues.HasQueuePair(app.Namespace) {
	// 	logCtx.Debugf("Creating new queue pair for namespace %s", app.Namespace)
	// 	err := s.queues.Create(app.Namespace)
	// 	if err != nil {
	// 		logCtx.Errorf("could not create new queue pair for namespace %s: %v", app.Namespace, err)
	// 		return
	// 	}
	// }
	if !s.queues.HasQueuePair(app.Namespace) {
		logCtx.Tracef("no agent connected to namespace %s, discarding", app.Namespace)
		return
	}
	q := s.queues.SendQ(app.Namespace)
	if q == nil {
		logCtx.Errorf("Help! queue pair for namespace %s disappeared!", app.Namespace)
		return
	}

	q.Add(app)
	logCtx.Tracef("Added app %s to send queue, total length now %d", app.QualifiedName(), q.Len())
}
