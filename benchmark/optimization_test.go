package benchmark

import "testing"

const (
	m1 = 0x5555555555555555
	m2 = 0x3333333333333333
	m4 = 0x0f0f0f0f0f0f0f0f
)

func calculate(x uint64) uint64 {
	x -= (x >> 1) & m1
	x = (x & m2) + ((x >> 2) & m2)
	return (x + (x >> 4)) & m4
}

func BenchmarkCalculate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculate(uint64(i))
	}
}

func BenchmarkCalculateEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// empty body
	}
}
