package queue

import (
	"fmt"
	"sync"
	"sync/atomic"

	"k8s.io/client-go/util/workqueue"
)

type queuepair struct {
	recvq     workqueue.RateLimitingInterface
	sendq     workqueue.RateLimitingInterface
	consumers atomic.Int32
}

type SendRecvQueues struct {
	queues    map[string]*queuepair
	queuelock sync.RWMutex
}

func NewSendRecvQueues() *SendRecvQueues {
	return &SendRecvQueues{
		queues: make(map[string]*queuepair),
	}
}

// HasQueuePair retruns true if a queue pair with name currently exists
func (q *SendRecvQueues) HasQueuePair(name string) bool {
	q.queuelock.RLock()
	defer q.queuelock.RUnlock()
	_, ok := q.queues[name]
	return ok
}

// Len returns the number of queue pairs held by q
func (q *SendRecvQueues) Len() int {
	q.queuelock.RLock()
	defer q.queuelock.RUnlock()
	return len(q.queues)
}

// RecvQ will return the send queue from the queue pair named name. If no such
// queue pair exists, returns nil
func (q *SendRecvQueues) SendQ(name string) workqueue.RateLimitingInterface {
	q.queuelock.RLock()
	defer q.queuelock.RUnlock()
	qp, ok := q.queues[name]
	if ok {
		return qp.sendq
	}
	return nil
}

// RecvQ will return the receive queue from the queue pair named name. If no
// such queue pair exists, returns nil
func (q *SendRecvQueues) RecvQ(name string) workqueue.RateLimitingInterface {
	q.queuelock.RLock()
	defer q.queuelock.RUnlock()
	qp, ok := q.queues[name]
	if ok {
		return qp.recvq
	}
	return nil
}

// Create creates and initializes a queue pair with name, and adds it to the
// list of available queues. The given name must be unique, if a queue pair
// with the same name already exists, Create will return an error.
func (q *SendRecvQueues) Create(name string) error {
	q.queuelock.RLock()
	defer q.queuelock.RUnlock()
	_, ok := q.queues[name]
	if ok {
		return fmt.Errorf("cannot initialize queue for %s: queue already exists", name)
	}
	qp := &queuepair{}
	qp.sendq = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "sendqueue")
	qp.recvq = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "recvqueue")
	q.queues[name] = qp

	return nil
}

// Delete will delete the named queue pair from the list of available queue.
// pairs. If shutdown is true, the Shutdown function will be called on both
// send and receive queues. If the named queue does not exist, Delete will
// return an error.
func (q *SendRecvQueues) Delete(name string, shutdown bool) error {
	q.queuelock.Lock()
	defer q.queuelock.Unlock()
	queue, ok := q.queues[name]
	if !ok {
		return fmt.Errorf("cannot drop queue %s: queue does not exist", name)
	}
	if shutdown {
		queue.recvq.ShutDown()
		queue.sendq.ShutDown()
	}
	delete(q.queues, name)
	return nil
}
