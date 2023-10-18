package agent

import (
	"io"
	"time"

	"github.com/jannfis/argocd-agent/internal/event"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/eventstreamapi"
	"github.com/sirupsen/logrus"
)

func (a *Agent) maintainConnection() error {
	go func() {
		var err error
		for {
			if !a.connected.Load() {
				err = a.remote.Connect(a.context, false)
				if err != nil {
					log().Warnf("Could not connect to %s: %v", a.remote.Addr(), err)
				} else {
					err = a.queues.Create(a.remote.ClientID())
					if err != nil {
						log().Warnf("Could not create agent queue pair: %v", err)
					} else {
						a.connected.Store(true)
					}
				}
			} else {
				a.handleStreamEvents()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return nil
}

func (a *Agent) handleStreamEvents() error {
	conn := a.remote.Conn()
	client := eventstreamapi.NewEventStreamClient(conn)
	stream, err := client.Subscribe(a.context)
	if err != nil {
		return err
	}
	syncCh := make(chan struct{})

	// Receive events from the subscription stream
	go func() {
		logCtx := log().WithFields(logrus.Fields{
			"module":    "StreamEvent",
			"direction": "Recv",
		})
		logCtx.Info("Starting to receive events from event stream")
		for a.connected.Load() {
			ev, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					close(syncCh)
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			logCtx.Infof("Received a new event from stream: %v", ev)
			switch event.EventType(ev.Event) {
			case event.EventTypeAddApp:
				err = a.createApplication(ev.Application)
				if err != nil {
					logCtx.Errorf("Error creating application: %v", err)
				}
			case event.EventTypeUpdateAppSpec:
				err = a.updateApplication(ev.Application)
				if err != nil {
					logCtx.Errorf("Error updating application: %v", err)
				}
			case event.EventTypeDeleteApp:
				err = a.deleteApplication(ev.Application)
				if err != nil {
					logCtx.Errorf("Error deleting application: %v", err)
				}
			default:
				logCtx.Infof("Unknown event received")
			}
		}
	}()

	for {
		select {
		case <-a.context.Done():
			return nil
		case <-syncCh:
			log().WithField("componet", "EventHandller").Info("Stream closed")
			return nil
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
