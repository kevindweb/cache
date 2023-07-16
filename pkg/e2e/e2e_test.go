package e2e

import (
	"sync"
	"testing"
)

var counter = 6379
var mu sync.Mutex

func BenchmarkEndToEnd(b *testing.B) {
	mu.Lock()
	counter++
	mycount := counter
	mu.Unlock()
	s := initializeBenchmark(mycount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(mycount)
	}
	b.StopTimer()
	s.Stop()
}
