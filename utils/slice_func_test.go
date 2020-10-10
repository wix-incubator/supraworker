package utils

import (
	"go.uber.org/goleak"
	"testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

var extraNumbers = 1

func BenchmarkContainsInts(b *testing.B) {
	benchmarkContainsIntsHelper(ContainsInts, b)
}

func benchmarkContainsIntsHelper(f func([]int, int) bool, b *testing.B) {
	in := make([]int, b.N+extraNumbers)
	for i := 0; i < b.N+extraNumbers; i++ {
		in[i] = i
	}
	n := 100
	if len(in) < n {
		n = len(in)
	}
	b.ResetTimer()
	for _, i := range in[:n] {
		if got := f(in, i+len(in)); got {
			b.Errorf("want %v, got %v", false, got)
		}
	}
	for _, i := range in[:n] {
		if got := f(in, i); !got {
			b.Errorf("want %v, got %v", true, got)
		}
	}
}
