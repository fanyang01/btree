package btree

import "sort"

type (
	children []interface{}
	keys     []interface{}

	Node struct {
		children children
		keys     keys
	}

	cmpFunc func(x, y interface{}) int
	Tree    struct {
		root            *Node
		b, height, size int
		cmp             cmpFunc
	}
)

func New(b int, f cmpFunc) *Tree {
	return &Tree{
		b:   b,
		cmp: f,
	}
}

// idx <= len(*s)
func (s *keys) insertAt(idx int, key interface{}) {
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

func (s *children) insertAt(idx int, c interface{}) {
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

func (t *Tree) newNode() *Node {
	return &Node{}
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

func (t *Tree) Lookup(k interface{}) (v interface{}, ok bool) {
	level, n := t.height, t.root
	if level == 0 {
		return
	}
	for ; level > 1; level-- {
		i := t.find(k, n)
		n = n.children[i].(*Node)
	}
	if i, found := t.findLeaf(k, n); found {
		return n.children[i], true
	}
	return
}

func (t *Tree) insert(n *Node, lv int, k, v interface{}) (kk, vv, old interface{}, split, replace bool) {
	if lv == 1 { // Leaf
		i, found := t.findLeaf(k, n)
		if found {
			old, replace = n.children[i], true
			n.children[i] = v
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
		n.keys.insertAt(i, k)
		n.children.insertAt(i+1, v)
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
	kk, vv, old, split, replace = t.insert(n.children[i].(*Node), lv-1, k, v)
	if !split {
		return
	}
	n.keys.insertAt(i, kk)
	n.children.insertAt(i+1, vv)
	if len(n.children) <= t.b {
		split = false
		return
	}
	kk, vv = t.split(n, len(n.keys)/2)
	return
}

func (t *Tree) Insert(k, v interface{}) (old interface{}, replace bool) {
	if t.height == 0 {
		x := t.newNode()
		x.keys = append(x.keys, k)
		x.children = append(x.children, nil, v)
		t.root, t.height = x, 1
		return
	}
	var kk, vv interface{}
	var split bool
	kk, vv, old, split, replace = t.insert(t.root, t.height, k, v)
	if !split {
		return
	}
	x := t.newNode()
	x.keys = append(x.keys, kk)
	x.children = append(x.children, t.root, vv)
	t.root, t.height = x, t.height+1
	return
}