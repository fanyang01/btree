package btree

type Pointer interface{}
type vT interface{}
type kT interface{}

type Store interface {
	// AllocNode allocates enough space for a B+ tree node and returns a reference to it.
	AllocNode() (ref Pointer, err error)
	// ReadNode reads the data pointed by ref and decodes it to a B+ tree node.
	ReadNode(ref Pointer) (*Node, error)
	// WriteNode writes encoded form of node n to space referenced by ref.
	WriteNode(ref Pointer, n *Node) error
	// DeallocNode deallocates space referenced by ref.
	DeallocNode(ref Pointer) error
	// AllocLeaf allocates space for a B+ tree leaf and returns a reference to it.
	AllocLeaf() (ref Pointer, err error)
	// ReadLeaf reads the data pointed by ref and decodes it to proper formats.
	ReadLeaf(ref Pointer) (vT, error)
	// WriteLeaf writes encoded form of v to space referenced by ref.
	WriteLeaf(ref Pointer, v vT) error
	// DeallocLeaf deallocates space referenced by ref.
	DeallocLeaf(ref Pointer) error
}

func (t *Tree) newNode() *Node {
	return t.pool.Get().(*Node)
}

func (t *Tree) freeNode(n *Node) {
	for i := range n.keys {
		n.keys[i] = nil
	}
	n.keys = n.keys[:0]
	for i := range n.children {
		n.children[i] = nil
	}
	n.children = n.children[:0]
	t.pool.Put(n)
}
