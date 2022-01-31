package html

// All nodes that also contain other nodes implment the NodeContainer interface.
type NodeContainer interface {
	Node
	Container
}

// Walk will do a depth-first traveresal through the html nodes. It will call
// the given fn for each node, unless fn returns false on the parent node.
func Walk(n Node, fn func(Node, []NodeContainer) bool) {
	walk(n, []NodeContainer{}, fn)
}

func walk(n Node, ps []NodeContainer, fn func(Node, []NodeContainer) bool) {
	if shouldWalkChildren := fn(n, ps); !shouldWalkChildren {
		return
	}

	nc, ok := n.(NodeContainer)
	if !ok {
		return
	}

	ps = append([]NodeContainer{nc}, ps...)
	for _, child := range nc.Children() {
		walk(child, ps, fn)
	}
}
