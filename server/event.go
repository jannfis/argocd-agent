package server

import (
	"context"
	"fmt"

	"github.com/jannfis/argocd-agent/internal/event"
	"golang.org/x/sync/semaphore"
	"k8s.io/client-go/util/workqueue"
)

func (s *Server) processAgentRecvQueue(ctx context.Context, agentName string, q workqueue.RateLimitingInterface) error {
	logCtx := log().WithField("module", "QueueProcessor").WithField("client", agentName)
	i, _ := q.Get()
	ev, ok := i.(*event.Event)
	if !ok {
		return fmt.Errorf("invalid data in queue: have:%T want:%T", i, ev)
	}
	logCtx.Debugf("Processing event: %s %s", ev.Type.String(), ev.Application.QualifiedName())
	switch ev.Type {
	case event.EventTypeUpdateAppStatus:
		err := s.appManager.UpdateStatus(ctx, ev.Application)
		if err != nil {
			return fmt.Errorf("could not update application status for %s: %w", ev.Application.QualifiedName(), err)
		}
	default:
		return fmt.Errorf("unable to process event of type %s", ev.Type.String())
	}
	return nil
}

// StartEventProcessor will start the event processor, which processes items
// from all queues as the items appear in the queues. Processing will be
// performed in parallel, and in the background, until the context ctx is done.
//
// If an error occurs before the processor could be started, it will be
// returned.
func (s *Server) StartEventProcessor(ctx context.Context) error {
	go func() {
		sem := semaphore.NewWeighted(s.options.eventProcessors)
		logCtx := log().WithField("module", "EventProcessor")
		for {
			for _, n := range s.queues.Names() {
				select {
				case <-ctx.Done():
					logCtx.Infof("Shutting down event processor")
					return
				default:
					// Though unlikely, the agent might have disconnected, and
					// the queue will be gone. In this case, we'll just skip.
					q := s.queues.RecvQ(n)
					if q == nil {
						logCtx.Debugf("Queue disappeared -- client probably has disconnected")
						continue
					}

					// Since q.Get() is blocking, we want to make sure something is actually
					// in the queue before we try to grab it.
					if q.Len() == 0 {
						continue
					}

					err := sem.Acquire(ctx, 1)
					if err != nil {
						return
					}
					go func(agentName string, q workqueue.RateLimitingInterface) {
						defer sem.Release(1)
						err := s.processAgentRecvQueue(ctx, agentName, q)
						if err != nil {
							logCtx.WithField("client", agentName).WithError(err).Errorf("Could not process agent recveiver queue")
						}
					}(n, q)
				}
			}
		}
	}()

	return nil
}
