package mutex

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkPlock(b *testing.B) {
	var (
		//m1 = make(map[int]int, 20000000)
		//m2 = make(map[int]int, 20000000)
		pl = NewPlock(context.Background())
	)
	b.Run("p", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			unlock := pl.Lock1(1)
			//	m1[i] = i
			unlock()
		}
	})

	b.Run("m", func(b *testing.B) {
		var pl = new(sync.Mutex)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pl.Lock()
			//m2[i] = i
			pl.Unlock()
		}
	})

	//b.Log("m1", len(m1))
	//b.Log("m2", len(m2))
}
