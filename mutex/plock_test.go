package mutex

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestUnlock(t *testing.T) {
	pl := NewPlock(context.Background())
	unlock := pl.Lock(1)
	n := unlock()
	fmt.Println(n)
	n = unlock()
	fmt.Println(n)
}

var pl = NewPlock(context.Background())

func TestPlock(t *testing.T) {
	total := 1000000
	t.Run("plock", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(total + 1)
		fn := func(p byte) {
			//fmt.Println(p, " -> take-lock")
			unlock := pl.Lock(p)
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
	/*
		t.Run("pqueue", func(t *testing.T) {
			pl := NewPlock(context.Background())
			heap.Push(pl, NewItem(1))
			fmt.Println("0", len(pl.Queue))
			v1 := heap.Pop(pl)
			fmt.Println("1", v1, len(pl.Queue))
			v2 := heap.Pop(pl)
			fmt.Println("2", v2, len(pl.Queue))
		})
	*/
}
