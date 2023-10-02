package batch

import (
	"io"
	"sync"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

type BatchEntry struct {
	app *v1alpha1.Application
	err error
}

type BatchQueue interface {
	Add(app *v1alpha1.Application)
	Len() int
	Has(app *v1alpha1.Application) bool
	Remove(app *v1alpha1.Application) error
	Empty() error
	Next() (*v1alpha1.Application, error)
	NextBatch() ([]*v1alpha1.Application, error)
}

var _ BatchQueue = &AutoBatchQueue{}

type AutoBatchCallback func(bq *AutoBatchQueue)

type AutoBatchQueue struct {
	entries map[string]BatchEntry
	lock    sync.RWMutex
	opts    AutoBatchOptions
}

type AutoBatchOptions struct {
	callback  AutoBatchCallback
	batchSize int
	interval  time.Duration
}

type AutoBatchOption func(o *AutoBatchOptions)

// WithBatchSize configures the batch size for a given batch queue
func WithBatchSize(batchSize int) AutoBatchOption {
	return func(o *AutoBatchOptions) {
		o.batchSize = batchSize
	}
}

// WithInterval configures the interval in which the batch callback will be executed
func WithInterval(interval time.Duration) AutoBatchOption {
	return func(o *AutoBatchOptions) {
		o.interval = interval
	}
}

// WithCallbackFunc sets the callback
func WithCallbackFunc(cb AutoBatchCallback) AutoBatchOption {
	return func(o *AutoBatchOptions) {
		o.callback = cb
	}
}

func NewAutoBatch(opts ...AutoBatchOption) *AutoBatchQueue {
	options := AutoBatchOptions{
		callback:  nil,
		batchSize: 0,
		interval:  0,
	}
	for _, o := range opts {
		o(&options)
	}
	bq := &AutoBatchQueue{
		entries: make(map[string]BatchEntry),
		opts:    options,
	}
	if options.interval > 0 && options.callback != nil {
		go func() {
			ticker := time.NewTicker(options.interval)
			for {
				select {
				case <-ticker.C:
					options.callback(bq)
					ticker.Reset(options.interval)
				default:
					time.Sleep(100 * time.Microsecond)
				}
			}
		}()
	}
	return bq
}

func (bq *AutoBatchQueue) Add(app *v1alpha1.Application) {
	bq.lock.Lock()
	bq.add(app)
	bq.lock.Unlock()
	if bq.opts.batchSize > 0 && len(bq.entries) >= bq.opts.batchSize && bq.opts.callback != nil {
		// Callback is executed in its own go routine
		go bq.opts.callback(bq)
	}
}

func (bq *AutoBatchQueue) Has(app *v1alpha1.Application) bool {
	bq.lock.RLock()
	defer bq.lock.RUnlock()
	_, ok := bq.entries[app.QualifiedName()]
	return ok
}

func (bq *AutoBatchQueue) add(app *v1alpha1.Application) {
	bqe, ok := bq.entries[app.QualifiedName()]
	if ok {
		bqe.app = app
		bqe.err = nil
	} else {
		bqe = BatchEntry{app: app}
		bq.entries[app.QualifiedName()] = bqe
	}
}

func (bq *AutoBatchQueue) Len() int {
	bq.lock.RLock()
	defer bq.lock.RUnlock()
	return len(bq.entries)
}

func (bq *AutoBatchQueue) Remove(app *v1alpha1.Application) error {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	_, ok := bq.entries[app.QualifiedName()]
	if !ok {
		return io.EOF
	} else {
		delete(bq.entries, app.QualifiedName())
	}
	return nil
}

func (bq *AutoBatchQueue) Empty() error {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	bq.entries = make(map[string]BatchEntry)
	return nil
}

func (bq *AutoBatchQueue) Next() (*v1alpha1.Application, error) {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	return bq.next()
}

func (bq *AutoBatchQueue) next() (*v1alpha1.Application, error) {
	for k, v := range bq.entries {
		delete(bq.entries, k)
		return v.app, nil
	}
	return nil, io.EOF
}

func (bq *AutoBatchQueue) NextBatch() ([]*v1alpha1.Application, error) {
	bq.lock.Lock()
	defer bq.lock.Unlock()
	var err error
	size := bq.opts.batchSize
	if len(bq.entries) < bq.opts.batchSize {
		size = len(bq.entries)
	}
	apps := make([]*v1alpha1.Application, size)
	for i := 0; i < bq.opts.batchSize; i++ {
		var app *v1alpha1.Application
		app, err = bq.next()
		if err != nil {
			break
		}
		apps[i] = app
	}
	if len(apps) == 0 || (err != nil && err != io.EOF) {
		return nil, err
	}
	return apps, nil
}
