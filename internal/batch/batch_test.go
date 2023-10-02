package batch

import (
	"io"
	"testing"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AddBatchQueueItem(t *testing.T) {
	t.Run("Add series of unique apps", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app2"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app3"}})
		assert.Equal(t, 3, bq.Len())
	})
	t.Run("Update same app", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1"}})
		assert.Equal(t, 1, bq.Len())
	})

	t.Run("Update app with similar name but different namespace", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "three"}})
		assert.Equal(t, 3, bq.Len())
	})
}

func Test_BatchRemoveItem(t *testing.T) {
	t.Run("Remove from empty batch queue", func(t *testing.T) {
		bq := NewAutoBatch()
		assert.ErrorIs(t, bq.Remove(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1"}}), io.EOF)
	})
	t.Run("Remove existing item", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}})
		assert.Equal(t, 2, bq.Len())
		assert.NoError(t, bq.Remove(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}}))
		assert.Equal(t, 1, bq.Len())
	})

	t.Run("Remove non-existing item", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}})
		assert.Equal(t, 2, bq.Len())
		assert.ErrorIs(t, bq.Remove(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "three"}}), io.EOF)
		assert.Equal(t, 2, bq.Len())
	})

}

func Test_BatchEmpty(t *testing.T) {
	t.Run("On empty queue", func(t *testing.T) {
		bq := NewAutoBatch()
		assert.Equal(t, 0, bq.Len())
		err := bq.Empty()
		assert.NoError(t, err)
		assert.Equal(t, 0, bq.Len())
	})

	t.Run("On populated queue", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		assert.Equal(t, 1, bq.Len())
		err := bq.Empty()
		assert.NoError(t, err)
		assert.Equal(t, 0, bq.Len())
	})
}

func Test_GetFromBatch(t *testing.T) {
	t.Run("Has item in queue", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		assert.True(t, bq.Has(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}}))
		assert.False(t, bq.Has(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app2", Namespace: "one"}}))
		assert.False(t, bq.Has(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}}))
	})
	t.Run("Get next item from empty queue", func(t *testing.T) {
		bq := NewAutoBatch()
		app, err := bq.Next()
		assert.Nil(t, app)
		assert.ErrorIs(t, err, io.EOF)
	})
	t.Run("Get next item from queue", func(t *testing.T) {
		bq := NewAutoBatch()
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		app, err := bq.Next()
		require.NotNil(t, app)
		require.NoError(t, err)
		assert.Equal(t, "app1", app.Name)
		assert.Equal(t, "one", app.Namespace)
		assert.Equal(t, 0, bq.Len())
		app, err = bq.Next()
		assert.Nil(t, app)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Get next item as batch from queue", func(t *testing.T) {
		bq := NewAutoBatch(WithBatchSize(2))
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app2", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app3", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app4", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app5", Namespace: "one"}})
		apps, err := bq.NextBatch()
		assert.NoError(t, err)
		assert.Len(t, apps, 2)
		assert.Equal(t, 3, bq.Len())
		apps, err = bq.NextBatch()
		assert.NoError(t, err)
		assert.Len(t, apps, 2)
		assert.Equal(t, 1, bq.Len())
		apps, err = bq.NextBatch()
		assert.NoError(t, err)
		assert.Len(t, apps, 1)
		assert.Equal(t, 0, bq.Len())
		apps, err = bq.NextBatch()
		assert.ErrorIs(t, err, io.EOF)
		assert.Len(t, apps, 0)
		assert.Equal(t, 0, bq.Len())
	})
}

func Test_Callback(t *testing.T) {
	t.Run("Callback with batch size", func(t *testing.T) {
		fn := func(bq *AutoBatchQueue) {
			bq.lock.RLock()
			defer bq.lock.RUnlock()
			if len(bq.entries) != bq.opts.batchSize {
				t.Error("callback should not have been called")
			}
		}
		bq := NewAutoBatch(WithBatchSize(2), WithCallbackFunc(fn))
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "two"}})
	})

	t.Run("Callback with interval on empty queue", func(t *testing.T) {
		called := false
		ch := make(chan bool)
		fn := func(bq *AutoBatchQueue) {
			_, err := bq.Next()
			ch <- err == io.EOF
		}
		NewAutoBatch(WithInterval(200*time.Millisecond), WithCallbackFunc(fn))
		timeout := time.NewTicker(2 * time.Second)
		select {
		case <-timeout.C:
			break
		case called = <-ch:
			break
		}
		assert.True(t, called, "callback not called within reasonable time")
	})

	t.Run("Callback with interval", func(t *testing.T) {
		called := false
		ch := make(chan bool)
		fn := func(bq *AutoBatchQueue) {
			bq.lock.RLock()
			defer bq.lock.RUnlock()
			if len(bq.entries) != 1 {
				t.Error("callback should not have been called")
				return
			}
			ch <- true
		}
		bq := NewAutoBatch(WithInterval(500*time.Millisecond), WithCallbackFunc(fn))
		bq.Add(&v1alpha1.Application{ObjectMeta: v1.ObjectMeta{Name: "app1", Namespace: "one"}})
		timeout := time.NewTicker(2 * time.Second)
		select {
		case <-timeout.C:
			break
		case called = <-ch:
			break
		}
		assert.True(t, called, "callback not called within reasonable time")
	})

}
