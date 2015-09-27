package btree

import (
	"math/rand"
	"testing"
	"time"
)

func compare(x, y interface{}) int {
	a, b := x.(int), y.(int)
	return a - b
}

func TestTree(t *testing.T) {
	N := 1 << 20
	b := New(4, compare)
	for i := 0; i < N; i++ {
		b.Insert(i, i)
	}
	for i := 0; i < N; i++ {
		if v, ok := b.Lookup(i); !ok {
			t.Errorf("Lookup %d failed\n", i)
		} else if v.(int) != i {
			t.Errorf("expected %d, got %d\n", i, v.(int))
		}
	}
	for i := N - 1; i >= 0; i-- {
		if v, ok := b.Remove(i); !ok {
			t.Errorf("Remove %d failed\n", i)
		} else if v.(int) != i {
			t.Errorf("expected %d, got %d\n", i, v.(int))
		}
	}
}

func TestRandom(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	N := 1 << 20
	b := New(4, compare)
	m := make(map[int]int)
	for i := 0; i < N; i++ {
		random := r.Intn(N)
		m[random] = random
		b.Insert(random, random)
	}
	for k, v := range m {
		if vv, ok := b.Lookup(k); !ok {
			t.Errorf("Lookup %d failed\n", k)
		} else if vv.(int) != v {
			t.Errorf("expected %d, got %d\n", v, vv.(int))
		}
	}
	for k, v := range m {
		if vv, ok := b.Remove(k); !ok {
			t.Errorf("Remove %d failed\n", k)
		} else if vv.(int) != v {
			t.Errorf("expected %d, got %d\n", v, vv.(int))
		}
	}
}
