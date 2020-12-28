package mutex

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Plock :
type Plock struct {
	tlock *sync.RWMutex
	cond  *sync.Cond
	p     int32
	s     int32
}

// NewPlock :
func NewPlock(_ctx context.Context) *Plock {
	pl := &Plock{
		tlock: new(sync.RWMutex),
		cond:  sync.NewCond(new(sync.Mutex)),
	}
	return pl
}

// RLock : read lock
func (p *Plock) RLock() {
	p.tlock.RLock()
}

// RUnlock : read unlock
func (p *Plock) RUnlock() {
	p.tlock.RUnlock()
}

// Unlock :
func (p *Plock) Unlock() {
	if atomic.CompareAndSwapInt32(&p.s, 1, 0) {
		atomic.StoreInt32(&p.p, 0)
	}
	p.tlock.Unlock()
	p.cond.L.Lock()
	p.cond.Broadcast()
	p.cond.L.Unlock()
}

// Lock0 :
func (p *Plock) Lock0() {
	if atomic.CompareAndSwapInt32(&p.p, 0, 1) {
		p.tlock.Lock()
		atomic.StoreInt32(&p.s, 1)
		return
	}
	p.cond.L.Lock()
	p.cond.Wait()
	p.cond.L.Unlock()
	p.Lock0()
}

// Lock :
func (p *Plock) Lock() {
	p.cond.L.Lock()
	if atomic.LoadInt32(&p.p) == 0 {
		p.cond.L.Unlock()
		p.tlock.Lock()
		return
	}
	p.cond.Wait()
	p.cond.L.Unlock()
	p.Lock()
}
