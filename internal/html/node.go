package html

// All node types implement the Node interface.
type Node interface {
	Parse(l Lexer) (Lexer, error)
}

// All types that contain nodes implement the Container intereface
type Container interface {
	Children() []Node
	AppendChild(Node)
}

// A Doc node represents a full html document.
type Doc struct {
	Roots []Node
}

func (n *Doc) Children() []Node {
	return n.Roots
}

func (n *Doc) AppendChild(child Node) {
	n.Roots = append(n.Roots, child)
}

// An ElNode represents an html element.
type ElNode struct {
	Tag        string
	ChildNodes []Node //TODO change to .ChildNodes

	//TODO, after parsing attr
	//Attrs []struct{ Name string; Dir string; value *TxtNode; }
}

func (n *ElNode) Children() []Node {
	return n.ChildNodes
}

func (n *ElNode) AppendChild(child Node) {
	n.ChildNodes = append(n.ChildNodes, child)
}

// A LeafElNode represents special cases of html elements who's children cannot be Parsed.
type LeafElNode struct {
	Tag     string
	Content string
}

// A TxtNode represents the plain text in the html.
type TxtNode struct {
	Content string

	//TODO, after parsing the {...}'s out
	//Tmpl []string
	//Args []string
}
