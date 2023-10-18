package server

import (
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/event"
)

// newAppCallback is a callback for the new app event
func (s *Server) newAppCallback(app *v1alpha1.Application) {
	logCtx := log().WithField("component", "NewAppCallback")
	if !s.queues.HasQueuePair(app.Namespace) {
		logCtx.Tracef("no agent connected to namespace %s, discarding", app.Namespace)
		return
	}
	q := s.queues.SendQ(app.Namespace)
	if q == nil {
		logCtx.Errorf("Help! queue pair for namespace %s disappeared!", app.Namespace)
		return
	}
	ev := event.Event{
		Type:        event.EventTypeAddApp,
		Application: app,
	}
	q.Add(ev)
	logCtx.Tracef("Added app %s to send queue, total length now %d", app.QualifiedName(), q.Len())
}

func (s *Server) updateAppCallback(old *v1alpha1.Application, new *v1alpha1.Application) {
	logCtx := log().WithField("component", "UpdateAppCallback")
	if !s.queues.HasQueuePair(old.Namespace) {
		logCtx.Tracef("no agent connected to namespace %s, discarding", old.Namespace)
		return
	}
	q := s.queues.SendQ(old.Namespace)
	if q == nil {
		logCtx.Errorf("Help! queue pair for namespace %s disappeared!", old.Namespace)
		return
	}
	ev := event.Event{
		Type:        event.EventTypeUpdateAppSpec,
		Application: new,
	}
	q.Add(ev)
	logCtx.Tracef("Added app %s to send queue, total length now %d", old.QualifiedName(), q.Len())
}

func (s *Server) deleteAppCallback(app *v1alpha1.Application) {
	logCtx := log().WithField("component", "DeleteAppCallback")
	if !s.queues.HasQueuePair(app.Namespace) {
		logCtx.Tracef("no agent connected to namespace %s, discarding", app.Namespace)
		return
	}
	q := s.queues.SendQ(app.Namespace)
	if q == nil {
		logCtx.Errorf("Help! queue pair for namespace %s disappeared!", app.Namespace)
		return
	}
	ev := event.Event{
		Type:        event.EventTypeDeleteApp,
		Application: app,
	}
	q.Add(ev)
	logCtx.WithField("event", "DeletaApp").WithField("sendq_len", q.Len()).Tracef("Added event to send queue")
}
