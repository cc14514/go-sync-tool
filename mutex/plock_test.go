package mutex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

var pl = NewPlock(context.Background())

func TestRLock(t *testing.T) {
	m := make(map[string]int)
	stopCh := make(chan struct{})
	go func() {
		defer func() {
			fmt.Println(time.Now(), "[test] --> Read Done.")
		}()
		fn := func(i int) {
			pl.RLock()
			rmap := make(map[string]int)
			for k, v := range m {
				b := strings.Split(k, "-")[0]
				rmap[b] = rmap[b] + v
			}
			pl.RUnlock()
			fmt.Println(time.Now(), "[test] --> Read : ", i, rmap)
		}
		for j := 0; j < 10; j++ {
			time.Sleep(100 * time.Millisecond)
			select {
			case <-stopCh:
				fn(j)
				return
			default:
				fn(j)
			}
		}
	}()

	wg := new(sync.WaitGroup)
	// write
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(j int) {
			defer func() {
				wg.Done()
				fmt.Println(time.Now(), "[test] --> Write Done.", j)
			}()
			for n := 0; n < 1000; n++ {
				//fmt.Println("-------------->", j, n)
				pl.Lock0()
				//fmt.Println("-------------------------------------<", j, n)
				m[fmt.Sprintf("%d-%d", j, n)] = n
				pl.Unlock()
			}
		}(i)
	}
	wg.Wait()
	close(stopCh)
	fmt.Println(time.Now(), "[test] --> Done.")
	time.Sleep(1 * time.Second)
}

func TestUnlock(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(3)
	rfn := func(k int) {
		fmt.Println(time.Now(), "[test] --> Try RLock: ", k)
		pl.RLock()
		fmt.Println(time.Now(), "[test] --> Catch RLock: ", k)
		time.Sleep(2 * time.Second)
		pl.RUnlock()
		fmt.Println(time.Now(), "[test] --> RUnlock : ", k)
		wg.Done()
	}
	go rfn(1)
	time.Sleep(100 * time.Microsecond)
	pl.Lock()
	go rfn(3)
	go rfn(2)
	fmt.Println(time.Now(), "[test] -------> lock")
	time.Sleep(2 * time.Second)
	pl.Unlock()
	fmt.Println(time.Now(), "[test] -------> unlock")
	wg.Wait()
}

func TestPlock(t *testing.T) {
	total := 1000000
	t.Run("plock", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(total + 1)
		fn := func(p byte) {
			//fmt.Println(p, " -> take-lock")
			pl.Lock()
			//fmt.Println(p, " -> locked")
			defer func() {
				pl.Unlock()
				//n := unlock()
				//fmt.Println("unlocked", ",", p, ",", n)
				wg.Done()
			}()
		}
		for i := total; i > 0; i-- {
			go fn(byte(2))
		}
		go fn(byte(0))
		wg.Wait()
		fmt.Println("Done.")
	})

	t.Run("slock", func(t *testing.T) {
		pl := new(sync.Mutex)
		wg := new(sync.WaitGroup)
		wg.Add(total)
		fn := func(p byte) {
			//fmt.Println(p, " -> take-lock")
			pl.Lock()
			//fmt.Println(p, " -> locked")
			defer func() {
				pl.Unlock()
				wg.Done()
				//fmt.Println("unlocked", ",", p, ",", n)
			}()
		}
		for i := total; i > 0; i-- {
			go fn(byte(i))
		}
		wg.Wait()
		fmt.Println("Done.")
	})
}

func TestAsync(t *testing.T) {
	wg := new(sync.WaitGroup)
	m := make(map[string]int)
	writer := func(i int) {
		defer func() {
			wg.Done()
			fmt.Println(time.Now(), "[test] <-- Writer Done: ", i)
		}()
		for j := 0; j < 100000; j++ {
			pl.Lock()
			k := fmt.Sprintf("t-%d", i)
			x := m[k]
			m[k] = x + j
			pl.Unlock()
		}
	}
	// reader
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			pl.RLock()
			fmt.Println(time.Now(), "[test] --> Read : ", m)
			pl.RUnlock()
			<-t.C
		}

	}()
	n := 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go writer(i)
	}
	wg.Wait()
	fmt.Println("Test Done.")
}

func TestAsync2(t *testing.T) {
	pl := new(sync.RWMutex)
	wg := new(sync.WaitGroup)
	m := make(map[string]int)
	writer := func(i int) {
		defer func() {
			wg.Done()
			fmt.Println(time.Now(), "[test] <-- Writer Done: ", i)
		}()
		for j := 0; j < 100000; j++ {
			pl.Lock()
			k := fmt.Sprintf("t-%d", i)
			x := m[k]
			m[k] = x + j
			pl.Unlock()
		}
	}
	// reader
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			pl.RLock()
			fmt.Println(time.Now(), "[test] --> Read : ", m)
			pl.RUnlock()
			<-t.C
		}

	}()
	n := 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go writer(i)
	}
	wg.Wait()
	fmt.Println("Test Done.")
}

/*
empty() 堆栈为空则返回真
pop() 移除栈顶元素（不会返回栈顶元素的值）
push() 在栈顶增加元素
size() 返回栈中元素数目
top() 返回栈顶元素 [1]
*/

func TestABStack(t *testing.T) {
	abq := NewABStack()
	abq.Push(newItem(0, uint(1)))
	abq.ForEach(func(i *Item) bool {
		fmt.Println("->", i.p, i.n)
		return true
	})
}

func TestFoo(t *testing.T) {
	var p chan int

	p = make(chan int)
	n1 := &p
	fmt.Println("Foobar", n1)
	a1 := unsafe.Pointer(n1)
	b := uintptr(a1)
	fmt.Println("1=======>", b)
	fmt.Println("2=======>", uint(b))

}
