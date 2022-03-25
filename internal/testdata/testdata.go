package testdata

type Pair[K, V any] struct {
	k K
	v V
}

func (p Pair[K, V]) Key() K {
	return p.k
}
