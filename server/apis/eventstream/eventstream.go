package eventstream

import (
	"fmt"
	"io"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/queue"
	"github.com/jannfis/argocd-agent/pkg/api/grpc/eventstreamapi"
	"github.com/jannfis/argocd-agent/pkg/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	eventstreamapi.UnimplementedEventStreamServer

	options *ServerOptions
	queues  *queue.SendRecvQueues
}

type ServerOptions struct {
	// recvQueue map[string]workqueue.RateLimitingInterface
	// sendQueue map[string]workqueue.RateLimitingInterface
}

type ServerOption func(o *ServerOptions)

// WithRecvQueue sets the receive queue for this server. The server will put
// all updates it receives from the client stream into this queue for further
// processing outside of this package.
// func WithRecvQueue(q map[string]workqueue.RateLimitingInterface) ServerOption {
// 	return func(o *ServerOptions) {
// 		o.recvQueue = q
// 	}
// }

// WithSendQueue sets the send queue for this server. The send queue holds all
// items that the server should stream to the client.
// func WithSendQueue(q map[string]workqueue.RateLimitingInterface) ServerOption {
// 	return func(o *ServerOptions) {
// 		o.sendQueue = q
// 	}
// }

// NewServer returns a new AppStream server instance with the given options
func NewServer(queues *queue.SendRecvQueues, opts ...ServerOption) *server {
	options := &ServerOptions{}
	for _, o := range opts {
		o(options)
	}
	return &server{
		queues:  queues,
		options: options,
	}
}

// Subscribe implements a bi-directional stream to exchange application updates
// between the agent and the server.
//
// The connection is kept open until the agent closes it, and the stream tries
// to send updates to the agent as long as possible.
func (s *server) Subscribe(sc eventstreamapi.EventStream_SubscribeServer) error {
	recvWaitC := make(chan struct{})
	sendWaitC := make(chan struct{})

	logCtx := log().WithField("method", "Subscribe")

	agentName, ok := sc.Context().Value(types.ContextAgentIdentifier).(string)
	if !ok {
		return fmt.Errorf("cannot process connection: invalid context: no agent name")
	}

	logCtx = logCtx.WithField("client", agentName)
	logCtx.Debug("A new client connected to the event stream")

	// We receive events in a dedicated go routine
	go func() {
		logCtx := logCtx.WithField("direction", "recv")
		for {
			logCtx.Tracef("Waiting to receive from channel")
			u, err := sc.Recv()
			if err == io.EOF {
				logCtx.Tracef("Remote end hung up")
				close(recvWaitC)
				return
			}
			// TODO: How to handle non-EOF errors?
			if err != nil {
				st, ok := status.FromError(err)
				if !ok || (st.Code() != codes.DeadlineExceeded && st.Code() != codes.Canceled) {
					logCtx.WithError(err).Error("Error receiving application update")
				}
				close(recvWaitC)
				return
			}
			logCtx.Infof("Received update for application %v (%p)", u.Application.QualifiedName(), u.Application)
			q := s.queues.RecvQ(agentName)
			if q == nil {
				logCtx.Warnf("I have no receive queue for agent")
				close(recvWaitC)
				return
			}
			q.Add(u.Application)
		}
	}()

	// We send events in a dedicated go routine
	go func() {
		logCtx := logCtx.WithField("direction", "send")
		logCtx.Tracef("Starting go routine in sending direction")
		for {
			select {
			case <-recvWaitC:
				logCtx.Info("Shutdown requested")
				close(sendWaitC)
				return
			case <-sc.Context().Done():
				logCtx.Info("Context canceled")
				close(sendWaitC)
				return
			default:
				q := s.queues.SendQ(agentName)
				if q == nil {
					logCtx.Warnf("I have no send queue for agent")
					close(sendWaitC)
					return
				}
				// Get() is blocking until there is at least one item in the
				// queue.
				logCtx.Tracef("Grabbing item from queue")
				item, shutdown := q.Get()
				if shutdown {
					logCtx.Tracef("Queue shutdown in progress")
					close(sendWaitC)
					return
				}
				logCtx.Tracef("Grabbed an item")
				if item == nil {
					return
				}

				app, ok := item.(*v1alpha1.Application)
				if !ok {
					logCtx.Warnf("invalid data in sendqueue")
					continue
				}

				// A Send() on the stream is actually not blocking.
				err := sc.Send(&eventstreamapi.Event{Application: app.DeepCopy()})
				// TODO: How to handle errors on send?
				if err != nil {
					logCtx.Errorf("Error sending data: %v", err)
					continue
				}
			}
		}
	}()

	<-recvWaitC
	return nil
}

// Push implements a client-side stream to receive updates for the client's
// Application resources.
func (s *server) Push(sub eventstreamapi.EventStream_PushServer) error {
	return nil
}

func log() *logrus.Entry {
	return logrus.WithField("module", "grpc.AppStream")
}
