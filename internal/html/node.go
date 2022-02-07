package html

// A NodeId will identify individual nodes in a html tree.
type NodeId int

// All node types implement the Node interface.
type Node interface {
	Id() NodeId
}

// All types that have a tag implement the Tager interface
type Tager interface {
	Tag() string
}

// All types that contain nodes implement the Container interface.
type Container interface {
	Children() []Node
}

type mutableContainer interface {
	Container
	appendChild(Node)
}

// All types that have string content implement the Tager interface.
type Contenter interface {
	Content() string
}

// A Doc node represents a full html document.
type Doc struct {
	id    NodeId
	roots []Node
}

func (n *Doc) Id() NodeId {
	return n.id
}

func (n *Doc) Children() []Node {
	return n.roots
}

func (n *Doc) appendChild(child Node) {
	n.roots = append(n.roots, child)
}

// An ElNode represents an html element.
type ElNode struct {
	id         NodeId
	tag        string
	childNodes []Node

	//TODO, after parsing attr
	//Attrs []struct{ Name string; Dir string; value *TxtNode; }
}

func (n *ElNode) Id() NodeId {
	return n.id
}

func (n *ElNode) Tag() string {
	return n.tag
}

func (n *ElNode) Children() []Node {
	return n.childNodes
}

func (n *ElNode) appendChild(child Node) {
	n.childNodes = append(n.childNodes, child)
}

// A LeafElNode represents special cases of html elements who's children
// cannot be Parsed.
type LeafElNode struct {
	id      NodeId
	tag     string
	content string
}

func (n *LeafElNode) Id() NodeId {
	return n.id
}

func (n *LeafElNode) Tag() string {
	return n.tag
}

func (n *LeafElNode) Content() string {
	return n.content
}

// A TxtNode represents the plain text in the html.
type TxtNode struct {
	id      NodeId
	content string
}

func (n *TxtNode) Id() NodeId {
	return n.id
}

func (n *TxtNode) Content() string {
	return n.content
}

// An ExprNode represents javascript expresion that value is put into a text node.
type ExprNode struct {
	id NodeId
	js string
}

func (n *ExprNode) Id() NodeId {
	return n.id
}

func (n *ExprNode) Content() string {
	return "{" + n.js + "}"
}

func (n *ExprNode) JsContent() string {
	return n.js
}

// IsContentWhiteSpace will check if all the .Content() only contains white
// space chars.
func IsContentWhiteSpace(n Contenter) bool {
	for _, c := range n.Content() {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f' {
			return false
		}
	}
	return true
}
