package btree

type Store interface {
	readNode(interface{}) *Node
	writeNode(*Node)
	readLeaf(interface{}) interface{}
	writeLeaf(interface{})
}
