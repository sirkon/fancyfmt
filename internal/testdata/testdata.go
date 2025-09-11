package testdata

// CFLAGS: -O2 -g
// #include "stdio"
import "C"
import (
	"encoding/json"
	"fmt"

	// Test comment
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
