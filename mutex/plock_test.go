package mutex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
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
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(j int) {
			defer func() {
				wg.Done()
				fmt.Println(time.Now(), "[test] --> Write Done.", j)
			}()
			for n := 0; n < 10; n++ {
				//fmt.Println("-------------->", j, n)
				if j%10 == 0 {
					pl.Lock0()
					fmt.Println("1111111111", j)
				} else {
					pl.Lock()
					fmt.Println("2222222222", j)
				}
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

func TestAB(t *testing.T) {
	m := make(map[string]uint64)
	fn := func(k string) {
		var i uint64 = 0
		for {
			pl.Lock()
			m[fmt.Sprintf("%s-%d", k, i)] = i
			pl.Unlock()
			i++
		}
	}
	for i := 0; i < 10000; i++ {
		go fn(fmt.Sprintf("t%d", i))
	}
	fmt.Println("-----------------------------------------------> lock")
	for i := 0; i < 6; i++ {
		time.Sleep(1 * time.Second)
		pl.Lock()
		fmt.Println(time.Now(), " -- ", i, len(m))
		pl.Unlock()
	}

	fmt.Println("-----------------------------------------------> lock0")
	for i := 0; i < 6; i++ {
		time.Sleep(1 * time.Second)
		pl.Lock0()
		fmt.Println(time.Now(), " -- ", i, len(m))
		pl.Unlock()
	}
}
