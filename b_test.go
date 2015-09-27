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

func testOrder(t *testing.T, order, N int) {
	t.Logf("testing b = %d...", order)
	b := New(order, compare)
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
	for i := N - 1; i >= N/2; i-- {
		if v, ok := b.Remove(i); !ok {
			t.Errorf("Remove %d failed\n", i)
		} else if v.(int) != i {
			t.Errorf("expected %d, got %d\n", i, v.(int))
		}
	}
	for i := 0; i < N/2; i++ {
		if v, ok := b.Remove(i); !ok {
			t.Errorf("Remove %d failed\n", i)
		} else if v.(int) != i {
			t.Errorf("expected %d, got %d\n", i, v.(int))
		}
	}
}

func TestInOrder(t *testing.T) {
	N := 1 << 16
	// TODO: index out of range: testOrder(t, 3, N)
	testOrder(t, 4, N)
	testOrder(t, 5, N)
	testOrder(t, 10, N)
	testOrder(t, 50, N)
}

func testRandom(t *testing.T, order, N int) {
	t.Logf("randomly testing b = %d...", order)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := New(order, compare)
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

func TestRandom(t *testing.T) {
	N := 1 << 16
	testRandom(t, 4, N)
	testRandom(t, 5, N)
	testRandom(t, 20, N)
	testRandom(t, 100, N)
}
