package mutex

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
)

var nonce uint64

// Plock :
type Plock struct {
	Queue    []*Item
	smap     *sync.Map
	mmap     map[uint64]chan chan struct{}
	eventCh  chan struct{}
	ctx      context.Context
	cancelFn context.CancelFunc
	lock     *sync.Mutex
}

// UnlockFn :
type UnlockFn func() uint64

// Lock :
func (p *Plock) Lock(pri byte) UnlockFn {
	// 1
	p.lock.Lock()
	item := NewItem(pri)
	heap.Push(p, item)
	//fmt.Println("-->", pri, item.n, 1)
	// 2
	lockCh := make(chan chan struct{}, 1)
	p.mmap[item.n] = lockCh
	//p.smap.Store(item.n, lockCh)
	p.lock.Unlock()
	//fmt.Println("-->", pri, item.n, 2, "lockCh=", lockCh)
	var unlockCh chan struct{}
	// 3
	p.eventCh <- struct{}{}
	//fmt.Println("-->", pri, item.n, 3, "wait", "lockCh=", lockCh)
	// 4
	unlockCh = <-lockCh
	//fmt.Println("-->", pri, item.n, 4, "notify", "unlockCh", unlockCh)
	return func() uint64 {
		//fmt.Println("-->", pri, "unlock")
		close(unlockCh)
		return item.n
	}
}

// NewPlock :
func NewPlock(_ctx context.Context) *Plock {
	slot := 200000
	ctx, cancelFn := context.WithCancel(_ctx)
	pl := &Plock{
		Queue:    make([]*Item, 0),
		smap:     new(sync.Map),
		mmap:     make(map[uint64]chan chan struct{}, slot),
		eventCh:  make(chan struct{}, slot*2),
		ctx:      ctx,
		cancelFn: cancelFn,
		lock:     new(sync.Mutex),
	}
	go pl.eventHandler()
	return pl
}

func (p *Plock) disp(i *Item) {
	// 7
	v, ok := p.smap.LoadAndDelete(i.n)
	//fmt.Println("-->", i.p, i.n, 7, ok, v)
	if !ok {
		panic("TODO : load can not be fail")
	}
	unlockCh := make(chan struct{})
	// 8
	v.(chan chan struct{}) <- unlockCh
	//fmt.Println("-->", i.p, i.n, 8, "lockCh <- unlockCh :", "unlockCh=", unlockCh, "lockCh=", v)
	<-unlockCh
	//fmt.Println("-->", i.p, i.n, 6)
}

func (p *Plock) eventHandler() {
	for {
		select {
		case <-p.eventCh:
			// 5
			p.lock.Lock()
			o := heap.Pop(p)
			i := o.(*Item)
			v, ok := p.mmap[i.n]
			if !ok {
				panic("TODO : load can not be fail")
			}
			delete(p.mmap, i.n)
			p.lock.Unlock()
			unlockCh := make(chan struct{})
			v <- unlockCh
			<-unlockCh
			//fmt.Println("eventHandler-loop", i)
			//fmt.Println("-->", i.p, i.n, 5)
			// 6
			//p.disp(i)
		case <-p.ctx.Done():
			return
		}
	}
}

// NewItem :
func NewItem(p byte) *Item {
	return &Item{p, atomic.AddUint64(&nonce, 1)}
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
	p.Queue, v = p.Queue[:len(p.Queue)-1], p.Queue[len(p.Queue)-1]
	return
}

// Len is the number of elements in the collection.
func (p *Plock) Len() int {
	return len(p.Queue)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (p *Plock) Less(i int, j int) bool {
	return p.Queue[i].p < p.Queue[j].p
}

// Swap swaps the elements with indexes i and j.
func (p *Plock) Swap(i int, j int) {
	p.Queue[i], p.Queue[j] = p.Queue[j], p.Queue[i]
}
