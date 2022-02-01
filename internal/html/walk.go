package html

// All nodes that contain other nodes implements the NodeContainer interface.
type NodeContainer interface {
	Node
	Container
}

// Walk will do a preorder depth-first traversal through the html nodes.
func Walk(n Node, fn func(Node, []NodeContainer) (bool, error)) error {
	err := walk(n, []NodeContainer{}, fn)
	if _, wasStopped := err.(stopWalkError); !wasStopped {
		return err
	}
	return nil
}

func walk(n Node, ps []NodeContainer, fn func(Node, []NodeContainer) (bool, error)) error {
	shouldWalkChildren, err := fn(n, ps)
	if err != nil {
		return err
	}
	if !shouldWalkChildren {
		return nil
	}

	nc, ok := n.(NodeContainer)
	if !ok {
		return nil
	}

	ps = append([]NodeContainer{nc}, ps...)
	for _, child := range nc.Children() {
		walk(child, ps, fn)
	}
	return nil
}

type stopWalkError struct{}

func (_ stopWalkError) Error() string {
	return "A Walk call has been stopped"
}

// StopWalk will stop traversal through the node tree without returning an
// error from Walk.
func StopWalk() (bool, stopWalkError) {
	return false, stopWalkError{}
}
