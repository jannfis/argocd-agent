package mock

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-application-agent/pkg/api/grpc/eventstreamapi"
	"google.golang.org/grpc"
)

// SendHook is a function that will be executed for the Send call in the mock
type SendHook func(s *MockEventServer, sub *eventstreamapi.Event) error

// RecvHook is a function that will be executed for the Recv call in the mock
type RecvHook func(s *MockEventServer) error

// MockEventServer implements a mock for the SubscriptionServer stream
// used for testing.
type MockEventServer struct {
	grpc.ServerStream

	NumSent     atomic.Uint32
	MaxSend     int
	NumRecv     atomic.Uint32
	MaxRecv     int
	BlockRecv   time.Duration
	RecvErr     error
	SendErr     error
	Application v1alpha1.Application
	RecvHooks   []RecvHook
	SendHooks   []SendHook
}

func (s *MockEventServer) AddSendHook(hook SendHook) {
	s.SendHooks = append(s.SendHooks, hook)
}

func (s *MockEventServer) AddRecvHook(hook RecvHook) {
	s.RecvHooks = append(s.RecvHooks, hook)
}

func (s *MockEventServer) Context() context.Context {
	return context.WithValue(context.TODO(), "agent_name", "default")
}

func (s *MockEventServer) Send(sub *eventstreamapi.Event) error {
	var err error
	for _, h := range s.SendHooks {
		if err = h(s, sub); err != nil {
			break
		}
	}
	if err == nil {
		s.NumSent.Add(1)
	}
	return err
}

func (s *MockEventServer) Recv() (*eventstreamapi.Event, error) {
	var err error
	for _, h := range s.RecvHooks {
		if err = h(s); err != nil {
			break
		}
	}
	if err == nil {
		s.NumRecv.Add(1)
		return &eventstreamapi.Event{Application: s.Application.DeepCopy()}, nil
	}

	return nil, err
}
