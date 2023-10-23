package namelock

import "sync"

type NameLock struct {
	l          sync.RWMutex
	namedLocks map[string]*sync.RWMutex
}

func NewNameLock() *NameLock {
	return &NameLock{
		namedLocks: make(map[string]*sync.RWMutex),
	}
}

func (nl *NameLock) RLock(name string) {
	nl.lock(name).RLock()
}

func (nl *NameLock) RUnlock(name string) {
	nl.lock(name).RUnlock()
}

func (nl *NameLock) Lock(name string) {
	nl.lock(name).Lock()
}

func (nl *NameLock) Unlock(name string) {
	nl.lock(name).Unlock()
}

func (nl *NameLock) TryLock(name string) bool {
	return nl.lock(name).TryLock()
}

func (nl *NameLock) TryRLock(name string) bool {
	return nl.lock(name).TryRLock()
}

func (nl *NameLock) lock(name string) *sync.RWMutex {
	nl.l.Lock()
	defer nl.l.Unlock()
	if l, ok := nl.namedLocks[name]; ok {
		return l
	} else {
		l = new(sync.RWMutex)
		nl.namedLocks[name] = l
		return l
	}
}
