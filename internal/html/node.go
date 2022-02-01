package html

// A NodeId will identify individual nodes in a html tree.
type NodeId int

// All node types implement the Node interface.
type Node interface {
	Id() NodeId
}

// All types that contain nodes implement the Container interface.
type Container interface {
	Children() []Node
}

type mutableContainer interface {
	Container
	appendChild(Node)
}

// A Doc node represents a full html document.
type Doc struct {
	id    NodeId
	Roots []Node
}

func (n *Doc) Id() NodeId {
	return n.id
}

func (n *Doc) Children() []Node {
	return n.Roots
}

func (n *Doc) appendChild(child Node) {
	n.Roots = append(n.Roots, child)
}

// TODO: This should be removed once Compnoent type has been updated
func (n *Doc) AppendChild(child Node) {
	n.appendChild(child)
}

// An ElNode represents an html element.
type ElNode struct {
	id         NodeId
	Tag        string
	ChildNodes []Node

	//TODO, after parsing attr
	//Attrs []struct{ Name string; Dir string; value *TxtNode; }
}

func (n *ElNode) Id() NodeId {
	return n.id
}

func (n *ElNode) Children() []Node {
	return n.ChildNodes
}

func (n *ElNode) appendChild(child Node) {
	n.ChildNodes = append(n.ChildNodes, child)
}

// A LeafElNode represents special cases of html elements who's children cannot be Parsed.
type LeafElNode struct {
	id      NodeId
	Tag     string
	Content string
}

func (n *LeafElNode) Id() NodeId {
	return n.id
}

// A TxtNode represents the plain text in the html.
type TxtNode struct {
	id      NodeId
	Content string

	//TODO, after parsing the {...}'s out
	//Tmpl []string
	//Args []string
}

func (n *TxtNode) Id() NodeId {
	return n.id
}
