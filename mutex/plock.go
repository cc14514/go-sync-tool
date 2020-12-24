package mutex

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Plock :
type Plock struct {
	Queue        []*Item
	smap         *sync.Map
	mmap         map[uint64]chan chan struct{}
	eventCh      chan struct{}
	ctx          context.Context
	lock         *sync.Mutex
	tlock        *sync.RWMutex
	ws           int32
	rc           int32
	lastUnlockFn UnlockFn
	nonce        uint64
	rcond        *sync.Cond
}

// UnlockFn :
type UnlockFn func() uint64

// RLock : read lock
func (p *Plock) RLock() {
	defer func() {
		p.tlock.RLock()
		rc := atomic.AddInt32(&p.rc, 1)
		fmt.Println("rlock->", rc)
	}()
	if atomic.LoadInt32(&p.ws) < 1 {
		return
	}
	// blocked
	p.rcond.L.Lock()
	p.rcond.Wait()
	p.rcond.L.Unlock()
}

// RUnlock : read unlock
func (p *Plock) RUnlock() {
	defer p.tlock.RUnlock()
	if old := atomic.AddInt32(&p.rc, -1); old == 0 {
		fmt.Println("runlock<-", old)
		p.eventCh <- struct{}{}
	}
}

// Lock : default priority is 254
func (p *Plock) Lock() {
	p.lastUnlockFn = p.Lock1(254)
}

// Unlock :
func (p *Plock) Unlock() {
	if p.lastUnlockFn != nil {
		p.lastUnlockFn()
	}
}

// Lock1 : pri to set priority
func (p *Plock) Lock1(pri byte) UnlockFn {
	// 1 注册一个拿锁请求
	p.lock.Lock()
	item := newItem(pri, &p.nonce)
	heap.Push(p, item)
	// 2
	lockCh := make(chan chan struct{}, 1)
	p.mmap[item.n] = lockCh
	p.lock.Unlock()
	var unlockCh chan struct{}
	// 3 等读锁
	if atomic.LoadInt32(&p.ws) < 1 && atomic.LoadInt32(&p.rc) < 1 {
		atomic.SwapInt32(&p.ws, 1)
		p.eventCh <- struct{}{}
	}
	// 4 等待叫号
	unlockCh = <-lockCh
	// 5 到号，上锁
	p.tlock.Lock()
	return func() uint64 {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("unlock err :", err)
			}
		}()
		close(unlockCh)
		p.tlock.Unlock()
		return item.n
	}
}

// NewPlock :
func NewPlock(_ctx context.Context) *Plock {
	slot := 1000000
	pl := &Plock{
		ctx:     _ctx,
		Queue:   make([]*Item, 0),
		smap:    new(sync.Map),
		mmap:    make(map[uint64]chan chan struct{}, slot),
		eventCh: make(chan struct{}, slot),
		rcond:   sync.NewCond(new(sync.Mutex)),
		lock:    new(sync.Mutex),
		tlock:   new(sync.RWMutex),
	}
	for _, fn := range pl.eventHandler() {
		go fn()
	}
	return pl
}

func (p *Plock) eventHandler() []func() {
	disp := func() {
		//disp := func() {
		// 5
		if v, ok := func() (v chan chan struct{}, ok bool) {
			p.lock.Lock()
			defer p.lock.Unlock()
			if len(p.Queue) == 0 {
				// RLock disp
				p.rcond.L.Lock()
				p.rcond.Broadcast()
				p.rcond.L.Unlock()
				atomic.SwapInt32(&p.ws, 0)
				return
			}
			o := heap.Pop(p) // pop is a write opt so take mutex lock
			i := o.(*Item)
			v, ok = p.mmap[i.n]
			delete(p.mmap, i.n)
			return
		}(); ok {
			unlockCh := make(chan struct{})
			v <- unlockCh
			<-unlockCh
			if atomic.LoadInt32(&p.rc) < 1 {
				p.eventCh <- struct{}{}
			}
		}
	}
	return []func(){
		func() { // event handler
			for {
				select {
				case <-p.eventCh:
					disp()
				case <-p.ctx.Done():
					return
				}
			}
			/*
				}, func() {
				}, func() {
			*/
		},
	}
}

func newItem(p byte, nonce *uint64) *Item {
	return &Item{p, atomic.AddUint64(nonce, 1)}
}

// Item :
type Item struct {
	p byte   // priority : hi to low : 0~255
	n uint64 // nonce auto inc
}

// Push impl heap.Interface
func (p *Plock) Push(x interface{}) {
	p.Queue = append(p.Queue, x.(*Item))
}

// Pop impl heap.Interface
func (p *Plock) Pop() (v interface{}) {
	//p.Queue, v = p.Queue[1:len(p.Queue)], p.Queue[0] // head : low
	p.Queue, v = p.Queue[:len(p.Queue)-1], p.Queue[len(p.Queue)-1] // head : height
	return
}

// Len is the number of elements in the collection.
func (p *Plock) Len() int {
	return len(p.Queue)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (p *Plock) Less(i int, j int) bool {
	//return p.Queue[i].p > p.Queue[j].p // head : low
	return p.Queue[i].p < p.Queue[j].p // head : height
}

// Swap swaps the elements with indexes i and j.
func (p *Plock) Swap(i int, j int) {
	p.Queue[i], p.Queue[j] = p.Queue[j], p.Queue[i]
}
