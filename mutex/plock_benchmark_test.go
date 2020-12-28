package mutex

import (
	"context"
	"sync"
	"testing"
)

func BenchmarkPlock(b *testing.B) {
	var (
		pl = NewPlock(context.Background())
	)
	b.Run("p", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%2 == 0 {
				pl.Lock0()
			} else {
				pl.Lock()
			}
			pl.Unlock()
		}
	})

	b.Run("m", func(b *testing.B) {
		var pl = new(sync.Mutex)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pl.Lock()
			pl.Unlock()
		}
	})

}
