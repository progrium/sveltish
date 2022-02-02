package html

// All nodes that contain other nodes implements the NodeContainer interface.
type NodeContainer interface {
	Node
	Container
}

// Parents are a slice where the last node is parent of another node and the
// rest are the parent of the nodes before it.
type Parents []NodeContainer

func (ps Parents) Parent() (Node, bool) {
	size := len(ps)
	if size == 0 {
		return nil, false
	}

	return ps[size-1], true
}

func (ps Parents) Depth() int {
	return len(ps)
}

// Walk will do a preorder depth-first traversal through the html nodes.
func Walk(n Node, fn func(Node, Parents) (bool, error)) error {
	err := walk(n, Parents{}, fn)
	if _, wasStopped := err.(stopWalkError); !wasStopped {
		return err
	}
	return nil
}

func walk(n Node, ps Parents, fn func(Node, Parents) (bool, error)) error {
	shouldWalkChildren, err := fn(n, ps)
	if err != nil {
		if shouldWalkChildren {
			panic("Cannot walk children once an error is returned, always return false with an error")
		}
		return err
	}
	if !shouldWalkChildren {
		return nil
	}

	nc, ok := n.(NodeContainer)
	if !ok {
		return nil
	}

	ps = append(ps, nc)
	for _, child := range nc.Children() {
		err = walk(child, ps, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

type stopWalkError struct{}

func (_ stopWalkError) Error() string {
	return "A html tree Walk was stopped"
}

// StopWalk will stop traversal through the node tree without returning an
// error from Walk.
func StopWalk() (bool, stopWalkError) {
	return false, stopWalkError{}
}
