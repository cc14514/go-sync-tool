package mutex

import (
	"context"
	"sync"
	"unsafe"
)

// Plock :
type Plock struct {
	Stack   *ABStack
	mmap    map[uint]chan chan struct{}
	eventCh chan struct{}
	ctx     context.Context
	lock    *sync.Mutex
	tlock   *sync.RWMutex
	//lastUnlockFn UnlockFn
	tcloseCh chan chan struct{}
}

// NewPlock :
func NewPlock(_ctx context.Context) *Plock {
	slot := 16
	pl := &Plock{
		ctx: _ctx,
		//Queue:   make([]*Item, 0),
		Stack:    NewABStack(),
		mmap:     make(map[uint]chan chan struct{}, slot),
		eventCh:  make(chan struct{}, slot*4096),
		lock:     new(sync.Mutex),
		tlock:    new(sync.RWMutex),
		tcloseCh: make(chan chan struct{}, 1),
	}

	for _, fn := range pl.eventHandler() {
		go fn()
	}
	return pl
}

// UnlockFn :
//type UnlockFn func() uint

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
	//p.lastUnlockFn()
	// TODO close unlockCh
	select {
	case uncloseCh := <-p.tcloseCh:
		p.tlock.Unlock()
		close(uncloseCh)
	default:
		panic("not locked")
	}

}

// Lock : default priority is 254
func (p *Plock) Lock() {
	p.lockFn(1)
}

// Lock0 :
func (p *Plock) Lock0() {
	p.lockFn(0)
}

// Lock1 : pri to set priority
func (p *Plock) lockFn(pri byte) {
	// 1 注册一个拿锁请求
	lockCh := make(chan chan struct{}, 1)
	n := uintptr(unsafe.Pointer(&lockCh))
	p.lock.Lock()
	item := newItem(pri, uint(n))
	p.Stack.Push(item)
	// 2
	p.mmap[item.n] = lockCh
	p.lock.Unlock()

	var unlockCh chan struct{}
	// 3 等通知
	p.eventCh <- struct{}{}
	unlockCh = <-lockCh
	// 到号，上锁
	p.tlock.Lock()
	p.tcloseCh <- unlockCh
	/*
		return func() uint {
			defer func() {
				if err := recover(); err != nil {
					fmt.Println("unlock err :", err)
				}
			}()
			close(unlockCh)
			p.tlock.Unlock()
			return item.n
		}
	*/
}

func (p *Plock) eventHandler() []func() {
	disp := func() {
		if v, ok := func() (v chan chan struct{}, ok bool) {
			p.lock.Lock()
			if p.Stack.Size() == 0 {
				p.lock.Unlock()
				return
			}
			i := p.Stack.Pop()
			v, ok = p.mmap[i.n]
			delete(p.mmap, i.n)
			p.lock.Unlock()
			return
		}(); ok {
			unlockCh := make(chan struct{})
			v <- unlockCh
			<-unlockCh
		} else {
			panic("????????")
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
		},
	}
}

func newItem(p byte, ptr uint) *Item {
	return &Item{p, ptr}
}

// Item :
type Item struct {
	p byte
	n uint
}

// ABStack :
type ABStack struct {
	a, b   byte
	stacka []*Item
	stackb []*Item
}

// NewABStack :
func NewABStack() *ABStack {
	return &ABStack{0, 1, make([]*Item, 0), make([]*Item, 0)}
}

// Push :
func (l *ABStack) Push(i *Item) {
	switch i.p {
	case l.a:
		l.stacka = append(l.stacka, i)
	case l.b:
		l.stackb = append(l.stackb, i)
	default:
		panic("error a / b")
	}

}

// Size :
func (l *ABStack) Size() int {
	return len(l.stacka) + len(l.stackb)
}

// Empty :
func (l *ABStack) Empty() bool {
	if l.Size() > 0 {
		return false
	}
	return true
}

// Pop :
func (l *ABStack) Pop() (v *Item) {
	if len(l.stacka) > 0 {
		return l.PopA()
	}
	return l.PopB()
}

// PopA :
func (l *ABStack) PopA() (v *Item) {
	if len(l.stacka) > 0 {
		l.stacka, v = l.stacka[:len(l.stacka)-1], l.stacka[len(l.stacka)-1]
	}
	return
}

// PopB :
func (l *ABStack) PopB() (v *Item) {
	if len(l.stackb) > 0 {
		l.stackb, v = l.stackb[:len(l.stackb)-1], l.stackb[len(l.stackb)-1]
	}
	return
}

// ForEach :
func (l *ABStack) ForEach(f func(*Item) bool) {
	for _, i := range append(l.stacka, l.stackb...) {
		if !f(i) {
			return
		}
	}
}
