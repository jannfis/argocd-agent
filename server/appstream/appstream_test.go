package appstream

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jannfis/argocd-application-agent/internal/queue"
	"github.com/jannfis/argocd-application-agent/server/appstream/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func Test_Subscribe(t *testing.T) {
	t.Run("Test send to subcription stream", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("test")
		s := NewServer(qs)
		st := &mock.MockSubscriptionServer{}
		st.AddRecvHook(func(s *mock.MockSubscriptionServer) error {
			ticker := time.NewTicker(500 * time.Millisecond)
			<-ticker.C
			return io.EOF
		})
		qs.SendQ("test").Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "test"}})
		qs.SendQ("test").Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "bar", Namespace: "test"}})
		err := s.Subscribe(st)
		assert.Nil(t, err)
		assert.Equal(t, 0, int(st.NumRecv.Load()))
		assert.Equal(t, 2, int(st.NumSent.Load()))
	})
	t.Run("Test recv from subscription stream", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("test")
		s := NewServer(qs)
		st := &mock.MockSubscriptionServer{MaxRecv: 2, Application: v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "test",
			},
		}}
		numReceived := 0
		st.AddRecvHook(func(s *mock.MockSubscriptionServer) error {
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
		assert.Equal(t, 2, qs.RecvQ("test").Len())
	})

	t.Run("Test connection closed by peer", func(t *testing.T) {
		qs := queue.NewSendRecvQueues()
		qs.Create("test")
		s := NewServer(qs)
		st := &mock.MockSubscriptionServer{RecvErr: fmt.Errorf("some error"), Application: v1alpha1.Application{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo",
				Namespace: "test",
			},
		}}
		st.AddRecvHook(func(s *mock.MockSubscriptionServer) error {
			return fmt.Errorf("some error")
		})
		err := s.Subscribe(st)
		assert.Nil(t, err)
		assert.Equal(t, 0, int(st.NumRecv.Load()))
		assert.Equal(t, 0, int(st.NumSent.Load()))
	})

}
