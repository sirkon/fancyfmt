package testdata

type Pair[
	K, V any,	C any,
] struct {
	k K
	v V
}


func (p Pair[K, V, T]) Key() K {
	return p.k
}
