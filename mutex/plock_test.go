package mutex

import (
	"context"
	"fmt"
	"net"
	"os"
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
		for n := 0; n < 5; n++ {
			go func() {
				i := 0
				for {
					select {
					case <-stopCh:
						fn(i)
						return
					default:
						fn(i)
					}
					i++
				}
			}()
		}
	}()

	wg := new(sync.WaitGroup)
	// write
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			defer func() {
				wg.Done()
				fmt.Println(time.Now(), "[test] --> Write Done.", j)
			}()
			time.Sleep(1 * time.Millisecond)
			for n := 0; n < 10000; n++ {
				pl.Lock()
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
	unlock := pl.Lock1(1)
	go rfn(3)
	go rfn(2)
	fmt.Println(time.Now(), "[test] -------> lock")
	time.Sleep(2 * time.Second)
	n := unlock()
	fmt.Println(time.Now(), "[test] -------> unlock", n)
	wg.Wait()
}

func TestPlock(t *testing.T) {
	total := 1000000
	t.Run("plock", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(total + 1)
		fn := func(p byte) {
			//fmt.Println(p, " -> take-lock")
			unlock := pl.Lock1(1)
			//fmt.Println(p, " -> locked")
			defer func() {
				unlock()
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

func TestFoo(t *testing.T) {
	addr := &net.UnixAddr{Name: "/tmp/a.sock"}
	addr2 := &net.UnixAddr{Name: "/tmp/b.sock"}
	proto := "unixgram"
	os.Remove(addr.Name)
	os.Remove(addr2.Name)
	sock, err := net.ListenUnixgram(proto, addr)
	if err != nil {
		panic(err)
	}

	reader := func(k string, sock *net.UnixConn) {
		buf := make([]byte, 8)
		i, err := sock.Read(buf)
		fmt.Println(k, err, i)
	}
	go reader("a", sock)
	go reader("b", sock)
	writer, err := net.DialUnix(proto, addr2, addr)
	if err != nil {
		panic(err)
	}
	i, err := writer.Write([]byte{1, 1, 1, 1, 1, 1, 1, 1})
	fmt.Println("write <-", i, err)
	time.Sleep(2 * time.Second)

}
