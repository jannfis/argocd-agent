package eventstream

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-agent/internal/queue"
	"github.com/jannfis/argocd-agent/server/apis/eventstream/mock"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func Test_Subscribe(t *testing.T) {
	t.Run("Test send to subcription stream", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("default")
		s := NewServer(qs)
		st := &mock.MockEventServer{AgentName: "default"}
		st.AddRecvHook(func(s *mock.MockEventServer) error {
			log().WithField("component", "RecvHook").Tracef("Entry")
			ticker := time.NewTicker(500 * time.Millisecond)
			<-ticker.C
			log().WithField("component", "RecvHook").Tracef("Exit")
			return io.EOF
		})
		qs.SendQ("default").Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "test"}})
		qs.SendQ("default").Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "bar", Namespace: "test"}})
		err := s.Subscribe(st)
		assert.Nil(t, err)
		assert.Equal(t, 0, int(st.NumRecv.Load()))
		assert.Equal(t, 2, int(st.NumSent.Load()))
	})
	t.Run("Test recv from subscription stream", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("default")
		s := NewServer(qs)
		st := &mock.MockEventServer{AgentName: "default", Application: v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
		}}
		numReceived := 0
		st.AddRecvHook(func(s *mock.MockEventServer) error {
			if numReceived >= 2 {
				return io.EOF
			}
			numReceived += 1
			return nil
		})
		err := s.Subscribe(st)
		assert.Nil(t, err)
		assert.Equal(t, 2, int(st.NumRecv.Load()))
		assert.Equal(t, 0, int(st.NumSent.Load()))
		assert.Equal(t, 2, qs.RecvQ("default").Len())
	})

	t.Run("Test connection closed by peer", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("default")
		s := NewServer(qs)
		st := &mock.MockEventServer{AgentName: "default"}
		st.AddRecvHook(func(s *mock.MockEventServer) error {
			return fmt.Errorf("some error")
		})
		err := s.Subscribe(st)
		assert.Nil(t, err)
		assert.Equal(t, 0, int(st.NumRecv.Load()))
		assert.Equal(t, 0, int(st.NumSent.Load()))
	})

}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}
