/*************************************************************************
 * Copyright (C) 2016-2019 PDX Technologies, Inc. All Rights Reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * @Time   : 2020/6/11 10:50 上午
 * @Author : liangc
 *************************************************************************/

package threadpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type asyncFn struct {
	fn   func(context.Context, []interface{})
	args []interface{}
}

func (f *asyncFn) apply(ctx context.Context) {
	f.fn(ctx, f.args)
}

type AsyncRunner struct {
	sync.Mutex
	wg                *sync.WaitGroup
	ctx               context.Context
	counter, min, max int32
	fnCh              chan *asyncFn
	closeCh           chan struct{}
	close             bool
	gc                time.Duration
}

func NewAsyncRunner(ctx context.Context, min, max int32) *AsyncRunner {
	wg := new(sync.WaitGroup)
	o := &AsyncRunner{
		ctx:     ctx,
		wg:      wg,
		counter: 0,
		min:     min,
		max:     max,
		fnCh:    make(chan *asyncFn),
		closeCh: make(chan struct{}),
		gc:      5 * time.Second,
	}
	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
		case <-o.closeCh:
		}
		wg.Done()
	}()
	return o
}

func (a *AsyncRunner) Size() int32 {
	return atomic.LoadInt32(&a.counter)
}

func (a *AsyncRunner) WaitClose() {
	a.close = true
	a.wg.Wait()
}

func (a *AsyncRunner) Wait() {
	a.wg.Wait()
}

func (a *AsyncRunner) spawn(tn int32, fn func(ctx context.Context, args []interface{}), args ...interface{}) {
	a.wg.Add(1)
	go func(tn int32) {
		defer a.wg.Done()
		timer := time.NewTimer(a.gc)
		for {
			select {
			case fn := <-a.fnCh:
				fn.apply(context.WithValue(a.ctx, "tn", tn))
			case <-a.ctx.Done():
				atomic.AddInt32(&a.counter, -1)
				return
			case <-timer.C:
				if func() bool {
					a.Lock()
					defer a.Unlock()
					if c := atomic.LoadInt32(&a.counter); c > a.min || a.close {
						c = atomic.AddInt32(&a.counter, -1)
						//fmt.Println("<--gc--", "tn", tn, a.close, "min", a.min, "counter", c)
						if c == 0 {
							close(a.closeCh)
						}
						return true
					}
					return false
				}() {
					return
				}
			}
			timer.Reset(a.gc)
		}
	}(tn)
}

func (a *AsyncRunner) Apply(fn func(ctx context.Context, args []interface{}), args ...interface{}) {
	select {
	case a.fnCh <- &asyncFn{fn, args}:
	default:
		//if tn := atomic.AddInt32(&a.counter, 1); tn <= a.max {
		a.Lock()
		tn := atomic.LoadInt32(&a.counter) + 1
		if tn <= a.max {
			atomic.AddInt32(&a.counter, 1)
			a.spawn(tn, fn, args)
		}
		a.Unlock()
		a.fnCh <- &asyncFn{fn, args}
	}
}

func Spawn(size int, fn func(int)) *sync.WaitGroup {
	wg := new(sync.WaitGroup)
	for i := 0; i < size; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fn(i)
		}(i)
	}
	return wg
}
