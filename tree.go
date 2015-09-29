package btree

import (
	"sort"
	"sync"
)

type (
	children []Pointer
	keys     []kT
)

type Node struct {
	ref      Pointer
	children children
	keys     keys
	t        *Tree
}

type cmpFunc func(x, y interface{}) int

type Tree struct {
	store        Store
	pool         sync.Pool
	root         Pointer
	b            int
	height, size int
	cmp          cmpFunc
}

func New(store Store, b int, f cmpFunc) *Tree {
	t := &Tree{
		store: store,
		b:     b,
		cmp:   f,
	}
	t.pool = sync.Pool{
		New: func() interface{} {
			return &Node{
				t: t,
			}
		},
	}
	return t
}

// idx <= len(*s)
func (s *keys) insertBefore(idx int, key interface{}) {
	if idx == len(*s) {
		*s = append(*s, key)
	} else {
		*s = append(*s, nil) // expand
		copy((*s)[idx+1:], (*s)[idx:])
		(*s)[idx] = key
	}
}

func (s *keys) removeAt(idx int) interface{} {
	v := (*s)[idx]
	copy((*s)[idx:], (*s)[idx+1:])
	*s = (*s)[:len(*s)-1] // shrink
	return v
}

func (t Tree) find(k interface{}, n *Node) (idx int) {
	return sort.Search(len(n.keys), func(i int) bool {
		return t.cmp(n.keys[i], k) > 0
	})
}

func (t Tree) findLeaf(k interface{}, n *Node) (idx int, found bool) {
	i := sort.Search(len(n.keys), func(i int) bool {
		return t.cmp(n.keys[i], k) >= 0
	})
	if i < len(n.keys) && t.cmp(n.keys[i], k) == 0 {
		return i + 1, true
	}
	return i, false
}

func (s *children) insertBefore(idx int, c interface{}) {
	if idx == len(*s) {
		*s = append(*s, c)
	} else {
		*s = append(*s, nil)
		copy((*s)[idx+1:], (*s)[idx:])
		(*s)[idx] = c
	}
}

func (s *children) removeAt(idx int) interface{} {
	c := (*s)[idx]
	copy((*s)[idx:], (*s)[idx+1:])
	*s = (*s)[:len(*s)-1]
	return c
}

/*
 *          +---+---+---+---+---+---+
 *          | k | k | k | k | k | k |
 *      +---+---+---+---+---+---+---+
 *      | p | v | v | v | v | v | v |
 *      +---+---+---+---+---+---+---+
 *                    x
 * -->
 *          +---+---+---+           +---+---+---+
 *          | k | k | k |           | k | k | k |
 *      +---+---+---+---+       +---+---+---+---+
 *      |   | v | v | v |       | p | v | v | v |
 *      +---+---+---+---+       +---+---+---+---+
 *        |     x                 ^     y
 *        |_______________________|
 */
func (t *Tree) splitLeaf(x *Node, before int) (key interface{}, y *Node) {
	i := before
	key = x.keys[i]
	y = t.newNode()
	y.keys = append(y.keys, x.keys[i:]...)
	x.keys = x.keys[:i]
	y.children = append(y.children, x.children[0])
	y.children = append(y.children, x.children[i+1:]...)
	x.children = x.children[:i+1]
	x.children[0] = y
	return
}

/*
 *      +---+---+---+---+---+---+---+
 *      | k | k | k | k | k | k | k |
 *      +---+---+---+---+---+---+---+
 *    +---+---+---+---+---+---+---+---+
 *    | p | p | p | p | p | p | p | p |
 *    +---+---+---+---+---+---+---+---+
 *                    x
 * -->
 *                     +---+
 *                     | k |
 *                     +---+
 *      +---+---+---+        +---+---+---+
 *      | k | k | k |        | k | k | k |
 *      +---+---+---+        +---+---+---+
 *    +---+---+---+---+    +---+---+---+---+
 *    | p | p | p | p |    | p | p | p | p |
 *    +---+---+---+---+    +---+---+---+---+
 *            x                    y
 */
func (t *Tree) split(x *Node, at int) (key interface{}, y *Node) {
	i := at
	key = x.keys[i]
	y = t.newNode()
	y.keys = append(y.keys, x.keys[i+1:]...)
	x.keys = x.keys[:i]
	y.children = append(y.children, x.children[i+1:]...)
	x.children = x.children[:i+1]
	return
}

func (t *Tree) Lookup(k interface{}) (v vT, ok bool, err error) {
	level := t.height
	if level == 0 {
		return
	}
	var n *Node
	if n, err = t.store.ReadNode(t.root); err != nil {
		return
	}
	for ; level > 1; level-- {
		i := t.find(k, n)
		if n, err = t.store.ReadNode(n.children[i]); err != nil {
			return
		}
	}
	if i, found := t.findLeaf(k, n); found {
		if v, err = t.store.ReadLeaf(n.children[i]); err != nil {
			return
		}
		ok = true
	}
	return
}

func (t *Tree) insert(n *Node, lv int, k, v interface{}) (kk kT, vv Pointer, old vT, split, replace bool, err error) {
	if lv == 1 { // Leaf
		i, found := t.findLeaf(k, n)
		if found {
			if old, err = t.store.ReadLeaf(n.children[i]); err != nil {
				return
			}
			if err = t.store.WriteLeaf(n.children[i], v); err == nil {
				replace = true
			}
			return
		}
		/*
		 *                         i
		 *           +---+---+---+---+---+---+
		 *           | k | k | k | k | k | k |
		 *       +---+---+---+---+---+---+---+
		 *       |   | v | v | v | v | v | v |
		 *       +---+---+---+---+---+---+---+
		 *                        i+1
		 * -->
		 *                            i
		 *      +---+---+---+ #===# +---+---+---+
		 *      | k | k | k | | k | | k | k | k |
		 *  +---+---+---+---+ +---+ +---+---+---+
		 *  |   | v | v | v | | v | | v | v | v |
		 *  +---+---+---+---+ #===# +---+---+---+
		 *                           i+1
		 */
		var ref Pointer
		if ref, err = t.store.AllocLeaf(); err != nil {
			return
		}
		if err = t.store.WriteLeaf(ref, v); err != nil {
			return
		}
		n.keys.insertBefore(i, k)
		n.children.insertBefore(i+1, ref)
		if len(n.keys) < t.b {
			return
		}
		split = true
		kk, vv = t.splitLeaf(n, len(n.keys)/2)
		return
	}
	/*
	 *                        i
	 *      +---+---+---+---+---+---+---+
	 *      | k | k | k | k | k | k | k |
	 *      +---+---+---+---+---+---+---+
	 *    +---+---+---+---+---+---+---+---+
	 *    | p | p | p | p | p | p | p | p |
	 *    +---+---+---+---+---+---+---+---+
	 *                      i  i+1
	 * -->
	 *                                i
	 *      +---+---+---+---+ #===# +---+---+---+
	 *      | k | k | k | k | | K | | k | k | k |
	 *      +---+---+---+---+ #===# +---+---+---+
	 *    +---+---+---+---+---+ #===# +---+---+---+
	 *    | p | p | p | p | p | | P | | p | p | p |
	 *    +---+---+---+---+---+ #===# +---+---+---+
	 *                      i          i+1
	 */
	i := t.find(k, n)
	var child *Node
	if child, err = t.store.ReadNode(n.children[i]); err != nil {
		return
	}
	kk, vv, old, split, replace, err = t.insert(child, lv-1, k, v)
	if err != nil {
		return
	}
	if err = t.store.WriteNode(child.ref, child); err != nil {
		return
	}
	if !split {
		return
	}
	n.keys.insertBefore(i, kk)
	n.children.insertBefore(i+1, vv)
	if len(n.children) <= t.b {
		split = false
		return
	}
	kk, vv = t.split(n, len(n.keys)/2)
	return
}

func (t *Tree) Insert(k, v interface{}) (old interface{}, replace bool, err error) {
	if t.height == 0 {
		x := t.newNode()
		x.keys = append(x.keys, k)
		var ref Pointer
		if ref, err = t.store.AllocLeaf(); err != nil {
			return
		}
		if err = t.store.WriteLeaf(ref, v); err != nil {
			return
		}
		x.children = append(x.children, nil, ref)
		if err = t.store.WriteNode(x.ref, x); err != nil {
			return
		}
		t.root = x
		t.height = 1
		return
	}
	var root *Node
	root, err = t.store.ReadNode(t.root)
	var kk, vv interface{}
	var split bool
	kk, vv, old, split, replace, err = t.insert(root, t.height, k, v)
	if err = t.store.WriteNode(root.ref, root); err != nil {
		return
	}
	if !split {
		return
	}
	x := t.newNode()
	x.keys = append(x.keys, kk)
	x.children = append(x.children, t.root, vv)
	if err = t.store.WriteNode(x.ref, x); err != nil {
		return
	}
	t.root = x
	t.height++
	return
}

/*
 *          +---+---+---+           +---+---+---+
 *          | k | k | k |           | k | k | k |
 *      +---+---+---+---+       +---+---+---+---+
 *      |   | v | v | v |       | p | v | v | v |
 *      +---+---+---+---+       +---+---+---+---+
 *              x                       y
 *
 * -->
 *          +---+---+---+---+---+---+
 *          | k | k | k | k | k | k |
 *      +---+---+---+---+---+---+---+
 *      | p | v | v | v | v | v | v |
 *      +---+---+---+---+---+---+---+
 *                    x
 */
func (x *Node) mergeNextLeaf(y, p *Node, yi int) {
	x.children = append(x.children, y.children[1:]...)
	x.keys = append(x.keys, y.keys...)
	x.children[0] = y.children[0]
	p.keys.removeAt(yi - 1)
	p.children.removeAt(yi)
	x.t.freeNode(y)
	// y.children = y.children[:0]
	// y.keys = y.keys[:0]
}

func (x *Node) borrowNextLeaf(y, p *Node, yi int) {
	x.keys = append(x.keys, y.keys.removeAt(0))
	x.children = append(x.children, y.children.removeAt(1))
	p.keys[yi-1] = y.keys[0]
}

func (y *Node) borrowPrevLeaf(x, p *Node, yi int) {
	y.keys.insertBefore(0, x.keys.removeAt(len(x.keys)-1))
	y.children.insertBefore(1, x.children.removeAt(len(x.children)-1))
	p.keys[yi-1] = y.keys[0]
}

/*
 *                           p   yi
 *                     +---+---+---+---+---+
 *                     |   | k1|   | k2|   |
 *                     +---+---+---+---+---+
 *                     /         \
 *                    /           \
 *             +---+---+---+ +---+---+---+
 *             |*1 | a |*2 | |*3 | b |*4 |
 *             +---+---+---+ +---+---+---+
 *                   x               y
 * -->
 *                           p
 *                     +---+---+---+
 *                     |   |k2 |   |
 *                     +---+---+---+
 *                     /         \
 *                    /           \
 *             +---+---+---+---+---+---+---+
 *             |*1 | a |*2 |k1 |*3 | b |*4 |
 *             +---+---+---+---+---+---+---+
 *                   x
 *
 */
func (x *Node) mergeNext(y, p *Node, yi int) {
	x.children = append(x.children, y.children...)
	x.keys = append(x.keys, p.keys[yi-1])
	x.keys = append(x.keys, y.keys...)
	p.keys.removeAt(yi - 1)
	p.children.removeAt(yi)
	// Clear y
	x.t.freeNode(y)
	// y.children = y.children[:0]
	// y.keys = y.keys[:0]
}

/*
 *                           p
 *                     +---+---+---+
 *                     |   | k |   |
 *                     +---+---+---+
 *                     /           \
 *                    /             \
 *             +---+---+---+   +---+---+---+---+---+
 *             |*1 | a |*2 |   |*3 | b |*4 | c |*5 |
 *             +---+---+---+   +---+---+---+---+---+
 *                   x               y
 * -->
 *                           p
 *                     +---+---+---+
 *                     |   | b |   |
 *                     +---+---+---+
 *                     /           \
 *                    /             \
 *        +---+---+---+---+---+   +---+---+---+
 *        |*1 | a |*2 | k |*3 |   |*4 | c |*5 |
 *        +---+---+---+---+---+   +---+---+---+
 *                   x               y
 *
 */
func (x *Node) borrowNext(y, p *Node, yi int) {
	x.keys = append(x.keys, p.keys[yi-1])
	x.children = append(x.children, y.children.removeAt(0))
	p.keys[yi-1] = y.keys.removeAt(0)
}

/*
 *                           p  yi
 *                     +---+---+---+
 *                     |   | k |   |
 *                     +---+---+---+
 *                     /           \
 *                    /             \
 *        +---+---+---+---+---+   +---+---+---+
 *        |*1 | a |*2 | b |*3 |   |*4 | c |*5 |
 *        +---+---+---+---+---+   +---+---+---+
 *                   x               y
 * -->
 *                           p
 *                     +---+---+---+
 *                     |   | b |   |
 *                     +---+---+---+
 *                     /           \
 *                    /             \
 *             +---+---+---+   +---+---+---+---+---+
 *             |*1 | a |*2 |   |*3 | k |*4 | c |*5 |
 *             +---+---+---+   +---+---+---+---+---+
 *                   x               y
 */
func (y *Node) borrowPrev(x, p *Node, yi int) {
	y.keys.insertBefore(0, p.keys[yi-1])
	y.children.insertBefore(0, x.children.removeAt(len(x.children)-1))
	p.keys[yi-1] = x.keys.removeAt(len(x.keys) - 1)
}

func (t *Tree) Remove(k interface{}) (v interface{}, found bool, err error) {
	if t.height == 0 {
		return
	}
	var root *Node
	root, err = t.store.ReadNode(t.root)
	v, found, err = t.remove(root, nil, t.height, 0, k)
	if err = t.store.WriteNode(root.ref, root); err != nil {
		return
	}
	if len(root.children) == 1 {
		n := root.children[0]
		t.store.DeallocNode(root)
		t.root = n
		t.height--
	}
	return
}

func (t *Tree) remove(n, p *Node, lv, pos int, k interface{}) (v interface{}, ok bool, err error) {
	if lv == 1 {
		i, found := t.findLeaf(k, n)
		if !found {
			return
		}
		ok = true
		ref := n.children[i]
		if v, err = t.store.ReadLeaf(ref); err != nil {
			return
		}
		if err = t.store.DeallocLeaf(ref); err != nil {
			return
		}
		n.keys.removeAt(i - 1)
		n.children.removeAt(i)
		if len(n.keys) >= t.b/2 || p == nil {
			return
		}
		switch {
		case pos < len(p.children)-1: // not the last child of parent
			next := p.children[pos+1].(*Node)
			if len(next.children) == t.b/2 {
				n.mergeNextLeaf(next, p, pos+1)
			} else {
				n.borrowNextLeaf(next, p, pos+1)
			}
		case pos == len(p.children)-1: // last child of parent
			prev := p.children[pos-1].(*Node)
			if len(prev.children) == t.b/2 {
				prev.mergeNextLeaf(n, p, pos)
			} else {
				n.borrowPrevLeaf(prev, p, pos)
			}
		default:
			panic("shouldn't get here")
		}
		return
	}
	// Internal node
	i := t.find(k, n)
	var child *Node
	if child, err = t.store.ReadNode(n.children[i]); err != nil {
		return
	}
	if v, ok, err = t.remove(child, n, lv-1, i, k); err != nil {
		return
	}
	t.store.WriteNode(child.ref, child)
	if len(n.children) >= t.b/2 || p == nil {
		return
	}
	switch {
	case pos < len(p.children)-1: // not the last child of parent
		next := p.children[pos+1].(*Node)
		if len(next.children) == t.b/2 {
			n.mergeNext(next, p, pos+1)
		} else {
			n.borrowNext(next, p, pos+1)
		}
	case pos == len(p.children)-1: // last child of parent
		prev := p.children[pos-1].(*Node)
		if len(prev.children) == t.b/2 {
			prev.mergeNext(n, p, pos)
		} else {
			n.borrowPrev(prev, p, pos)
		}
	default:
		panic("shouldn't get here")
	}
	return
}
