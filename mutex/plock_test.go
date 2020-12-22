package mutex

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestQueue(t *testing.T) {
	total := 100000
	t.Run("plock", func(t *testing.T) {
		pl := NewPlock(context.Background())
		wg := new(sync.WaitGroup)
		wg.Add(total)
		fn := func(p byte) {
			//fmt.Println(p, " -> take-lock")
			unlock := pl.Lock(p)
			//fmt.Println(p, " -> locked")
			defer func() {
				unlock()
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
