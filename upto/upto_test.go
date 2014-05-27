package upto

import "testing"

const innerLoops = 10000

func BenchmarkFor(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		n = i
	}
	var _ = n
}

func BenchmarkUpTo(b *testing.B) {
	n := 0
	for i := range UpTo(b.N) {
		n = i
	}
	var _ = n
}

func BenchmarkUnusedLoopVarUpTo(b *testing.B) {
	for _ = range UpTo(b.N) {
	}
}

func BenchmarkUnusedLoopVarFor(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}

func BenchmarkDoubleFor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < innerLoops; j++ {
		}
	}
}

func BenchmarkDoubleUpTo(b *testing.B) {
	for _ = range UpTo(b.N) {
		for _ = range UpTo(innerLoops) {
		}
	}
}

func BenchmarkFor_UpTo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _ = range UpTo(innerLoops) {
		}
	}
}

func BenchmarkUpTo_For(b *testing.B) {
	for _ = range UpTo(b.N) {
		for j := 0; j < innerLoops; j++ {
		}
	}
}
