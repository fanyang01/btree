package btree

import "testing"

func compare(x, y interface{}) int {
	a, b := x.(int), y.(int)
	return a - b
}

func TestTree(t *testing.T) {
	b := New(4, compare)
	for i := 0; i < 1<<10; i++ {
		b.Insert(i, i)
	}
	for i := 0; i < 1<<10; i++ {
		if v, ok := b.Lookup(i); !ok || v.(int) != i {
			t.Errorf("expected %d, got %d\n", i, v.(int))
		}
	}
}
