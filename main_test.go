package main

import (
	"testing"
)

func BenchmarkEndToEnd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark()
	}
	b.StopTimer()
}
