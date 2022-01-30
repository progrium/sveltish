package sveltish

type HTMLNode interface {
	Walk([]HTMLNode, func(HTMLNode, []HTMLNode) bool)
}

type HTMLNodeContainer interface {
	AppendChild(HTMLNode)
}

type RootNode struct {
	Roots []HTMLNode
}

func (n *RootNode) Walk(ps []HTMLNode, fn func(HTMLNode, []HTMLNode) bool) {
	walkChildren := fn(n, ps)
	if !walkChildren {
		return
	}

	ps = append([]HTMLNode{n}, ps...)
	for _, child := range n.Roots {
		child.Walk(ps, fn)
	}
}

func (n *RootNode) AppendChild(child HTMLNode) {
	n.Roots = append(n.Roots, child)
}

type ElNode struct {
	Tag string
	Els []HTMLNode

	//TODO, after parsing attr
	//Attrs []struct{ Name string; Dir string; value *TxtNode; }
}

func (n *ElNode) Walk(ps []HTMLNode, fn func(HTMLNode, []HTMLNode) bool) {
	walkChildren := fn(n, ps)
	if !walkChildren {
		return
	}

	ps = append([]HTMLNode{n}, ps...)
	for _, child := range n.Els {
		child.Walk(ps, fn)
	}
}

func (n *ElNode) AppendChild(child HTMLNode) {
	n.Els = append(n.Els, child)
}

type LeafElNode struct {
	Tag     string
	Content string
}

func (n *LeafElNode) Walk(ps []HTMLNode, fn func(HTMLNode, []HTMLNode) bool) {
	fn(n, ps)
}

type TxtNode struct {
	Content string

	//TODO, after parsing the {...}'s out
	//Tmpl []string
	//Args []string
}

func (n *TxtNode) Walk(ps []HTMLNode, fn func(HTMLNode, []HTMLNode) bool) {
	fn(n, ps)
}

func WalkHTML(n HTMLNode, fn func(n HTMLNode, ps []HTMLNode) bool) {
	n.Walk([]HTMLNode{}, fn)
}
