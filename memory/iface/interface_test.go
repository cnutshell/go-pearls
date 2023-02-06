package iface

import (
	"testing"
)

var global interface{}

func BenchmarkInterface(b *testing.B) {
	var local interface{}
	for i := 0; i < b.N; i++ {
		local = calculate(i) // assign value to interface{}
	}
	global = local
}

// values is bigger than single machine word.
type values struct {
	value  int
	double int
	triple int
}

func calculate(i int) values {
	return values{
		value:  i,
		double: i * 2,
		triple: i * 3,
	}
}
