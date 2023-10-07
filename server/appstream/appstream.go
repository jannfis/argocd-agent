package appstream

import (
	"io"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-application-agent/internal/queue"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/appstreamapi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	appstreamapi.UnimplementedAppStreamServer

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
func (s *server) Subscribe(sc appstreamapi.AppStream_SubscribeServer) error {
	waitc := make(chan struct{})
	logCtx := log().WithField("method", "Subscribe")

	agentName := "testagent"

	// Recv() is blocking, so we run the receiver part in its own go routine
	go func() {
		logCtx := logCtx.WithField("direction", "recv")
		for {
			u, err := sc.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			// TODO: How to handle non-EOF errors?
			if err != nil {
				st, ok := status.FromError(err)
				if !ok || (st.Code() != codes.DeadlineExceeded && st.Code() != codes.Canceled) {
					logCtx.WithError(err).Error("Error receiving application update")
				}
				close(waitc)
				return
			}
			logCtx.Infof("Received update for application %v (%p)", u.Application.QualifiedName(), u.Application)
			q := s.queues.RecvQ(agentName)
			if q == nil {
				logCtx.Warnf("I have no receive queue for agent")
				continue
			}
			q.Add(u.Application)
		}
	}()
	go func() {
		logCtx := logCtx.WithField("direction", "send")
		for {
			select {
			case <-waitc:
				logCtx.Info("Shutdown requested")
				return
			case <-sc.Context().Done():
				logCtx.Info("Context canceled")
				return
			default:
				q := s.queues.SendQ(agentName)
				if q == nil {
					logCtx.Warnf("I have no send queue for agent")
					continue
				}
				item, shutdown := q.Get()
				if shutdown {
					close(waitc)
					return
				}
				if item == nil {
					return
				}
				app, ok := item.(*v1alpha1.Application)
				if !ok {
					logCtx.Warnf("invalid data in sendqueue")
					continue
				}
				err := sc.Send(&appstreamapi.Subscription{Application: app.DeepCopy()})
				// TODO: How to handle errors on send?
				if err != nil {
					logCtx.Errorf("Error sending data: %v", err)
					continue
				}
			}
		}
	}()

	<-waitc
	return nil
}

// Push implements a client-side stream to receive updates for the client's
// Application resources.
func (s *server) Push(sub appstreamapi.AppStream_PushServer) error {
	return nil
}

func log() *logrus.Entry {
	return logrus.WithField("module", "grpc.AppStream")
}
