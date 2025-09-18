package testdata

// CFLAGS: -O2 -g
// #include "stdio"
import "C"

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirkon/errors"
)

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

func playground(a int, b string, c any) error {
	fmt.Println(
		a, b, // Do not touch.
		c,
	)
	payload := map[string]any{
		"a": 1, "b": b, // I don't want formatting.
		"c": c,
	}
	_, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal test payload")
	}

	fmt.Println([]byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08,
		0x09,
	})

	return nil
}

func init() {
	someFunc(
		context.Background(),
		map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		},
		"stringstringstringstring!",
		nil,
	)

	var _ = []byte{
		0x01, 0x02, 0x03, 0x04, 0x05,
		0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}

}

func someFunc(a ...any) {}
