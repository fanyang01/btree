package btree

type MemStore struct{}

func (m MemStore) AllocNode() (Pointer, error) {
	n := &Node{}
	n.ref = n
	return n.ref, nil
}

func (m MemStore) ReadNode(ref Pointer) (*Node, error) {
	return ref.(*Node), nil
}

func (m MemStore) WriteNode(ref Pointer, n *Node) error { return nil }

func (m MemStore) DeallocNode(ref Pointer) error { return nil }

func (m MemStore) AllocLeaf() (Pointer, error) {
	return new(int), nil
}

func (m MemStore) ReadLeaf(ref Pointer) (vT, error) {
	return *(ref.(*int)), nil
}

func (m MemStore) WriteLeaf(ref Pointer, v vT) error {
	*(ref.(*int)) = v.(int)
	return nil
}

func (m MemStore) DeallocLeaf(ref Pointer) error {
	return nil
}
