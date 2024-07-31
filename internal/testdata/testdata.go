package testdata

// CFLAGS: -O2 -g
// #include "stdio"
import "C"

type Pair[K, V any, C any] struct {
	k K
	v V
}

func Couple[K any, V any](k K, v V) (K, V) {
	return k, v
}

func (p Pair[K, V, T]) Key() K {
	return p.k
}
