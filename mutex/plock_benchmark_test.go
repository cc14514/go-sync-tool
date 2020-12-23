package mutex

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkPlock(b *testing.B) {
	var (
		m1 = make(map[int]int, 20000000)
		m2 = make(map[int]int, 20000000)
		pl = NewPlock(context.Background())
	)
	b.Run("p", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			p := byte(1)
			if i%30 == 0 {
				p = byte(0)
			}
			unlock := pl.Lock(p)
			m1[i] = i
			unlock()
		}
	})

	b.Run("m", func(b *testing.B) {
		var pl = new(sync.Mutex)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pl.Lock()
			m2[i] = i
			pl.Unlock()
		}
	})

	b.Log("m1", len(m1))
	b.Log("m2", len(m2))
}

/*
func BenchmarkFoo(b *testing.B) {
	num := 10
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%d\n", num)
	}
}
*/
