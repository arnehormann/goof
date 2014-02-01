package goof

import "testing"

func BenchmarkUpTo(b *testing.B) {
	n := 0
	b.ResetTimer()
	for i := range UpTo(b.N) {
		n = i
	}
	var _ = n
}

func BenchmarkFor(b *testing.B) {
	n := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n = i
	}
	var _ = n
}
