package server

import (
	"context"
	"fmt"
	"time"

	"github.com/jannfis/argocd-agent/internal/event"
	"github.com/jannfis/argocd-agent/internal/namelock"
	"github.com/jannfis/argocd-agent/pkg/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"k8s.io/client-go/util/workqueue"
)

// processRecvQueue processes an entry from the receiver queue, which holds the
// events received by agents. It will trigger updates of resources in the
// server's backend.
func (s *Server) processRecvQueue(ctx context.Context, agentName string, q workqueue.RateLimitingInterface) error {
	i, _ := q.Get()
	ev, ok := i.(*event.Event)
	if !ok {
		return fmt.Errorf("invalid data in queue: have:%T want:%T", i, ev)
	}

	agentMode := s.agentMode(agentName)
	incoming := ev.Application

	logCtx := log().WithFields(logrus.Fields{
		"module":   "QueueProcessor",
		"client":   agentName,
		"mode":     agentMode.String(),
		"event":    ev.Type.String(),
		"incoming": incoming.QualifiedName(),
	})

	logCtx.Debugf("Processing event")
	switch ev.Type {
	case event.EventAppAdded:
		if agentMode == types.AgentModeAutonomous {
			_, err := s.appManager.Create(ctx, ev.Application)
			if err != nil {
				return fmt.Errorf("could not create application %s: %w", ev.Application.QualifiedName(), err)
			}
		} else {
			logCtx.Debugf("Discarding event, because agent is not in autonomous mode")
			return nil
		}
	case event.EvenAppStatusUpdated:
		var err error
		if agentMode == types.AgentModeAutonomous {
			_, err = s.appManager.UpdateAutonomousApp(ctx, incoming)
		} else {
			err = fmt.Errorf("event type not allowed when mode is not autonomous")
		}
		if err != nil {
			return fmt.Errorf("could not update application status for %s: %w", incoming.QualifiedName(), err)
		}
		logCtx.Infof("Updated application status %s", incoming.QualifiedName())
	case event.EvenAppSpecUpdated:
		var err error
		if agentMode == types.AgentModeManaged {
			_, err = s.appManager.UpdateStatus(ctx, incoming)
		} else {
			err = fmt.Errorf("event type not allowed when mode is not managed")
		}
		if err != nil {
			return fmt.Errorf("could not update application status for %s: %w", incoming.QualifiedName(), err)
		}
		logCtx.Infof("Updated application spec %s", incoming.QualifiedName())
	default:
		return fmt.Errorf("unable to process event of type %s", ev.Type.String())
	}
	q.Done(ev)
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
		queueLock := namelock.NewNameLock()
		logCtx := log().WithField("module", "EventProcessor")
		for {
			for _, queueName := range s.queues.Names() {
				select {
				case <-ctx.Done():
					logCtx.Infof("Shutting down event processor")
					return
				default:
					// Though unlikely, the agent might have disconnected, and
					// the queue will be gone. In this case, we'll just skip.
					q := s.queues.RecvQ(queueName)
					if q == nil {
						logCtx.Debugf("Queue disappeared -- client probably has disconnected")
						break
					}

					// Since q.Get() is blocking, we want to make sure something is actually
					// in the queue before we try to grab it.
					if q.Len() == 0 {
						break
					}

					// We lock this specific queue, so that we won't process two
					// items of the same queue at the same time. Queues must be
					// processed in the right order.
					//
					// If it's not possible to get a lock (i.e. a lock is already
					// being held elsewhere), we continue with the next queue.
					if !queueLock.TryLock(queueName) {
						logCtx.Tracef("Could not acquire queue lock %s", queueName)
						break
					}

					logCtx.Trace("Acquired lock")

					err := sem.Acquire(ctx, 1)
					if err != nil {
						logCtx.Tracef("Error acquiring semaphore: %v", err)
						queueLock.Unlock(queueName)
						break
					}

					logCtx.Trace("Acquired semaphore")

					go func(agentName string, q workqueue.RateLimitingInterface) {
						defer func() {
							sem.Release(1)
							queueLock.Unlock(agentName)
						}()
						err := s.processRecvQueue(ctx, agentName, q)
						if err != nil {
							logCtx.WithField("client", agentName).WithError(err).Errorf("Could not process agent recveiver queue")
						}
					}(queueName, q)
				}
			}
			// Give the CPU a little rest when no agents are connected
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return nil
}
