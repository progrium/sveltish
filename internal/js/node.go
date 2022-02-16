package js

import (
	"strings"
	"unicode"
)

// All node types implement the Node interface.
type Node interface {
	Js() string
}

// A Script node represents a full js script tag.
type Script struct {
	roots []Node
}

func (n *Script) Js() string {
	js := ""
	for _, n := range n.roots {
		js += n.Js()
	}
	return js
}

func (n *Script) appendChild(child Node) {
	n.roots = append(n.roots, child)
}

// A CommentNode represents a js comment (for reprinting).
type CommentNode struct {
	content []byte
}

func (n *CommentNode) Js() string {
	return string(n.content)
}

// A childComments represents js comments inside another node (for reprinting).
type childComments struct {
	nodes []*CommentNode
}

func (cs *childComments) appendNil() {
	cs.nodes = append(cs.nodes, nil)
}

func (cs *childComments) appendChild(node *CommentNode) {
	cs.nodes = append(cs.nodes, node)
}

func (cs *childComments) pop() {
	cs.nodes = cs.nodes[:len(cs.nodes)-1]
}

func (cs *childComments) injectBetween(allData ...string) string {
	js := ""
	notSet := 0
	for i, data := range allData {
		if data == "" {
			notSet += 1
			continue
		}

		js += data

		for len(cs.nodes) <= i-notSet {
			cs.appendNil()
		}
		if n := cs.nodes[i-notSet]; n != nil {
			js += n.Js()
		}
	}
	return js
}

// A VarNode represents a js variable initlization/declarion.
type VarNode struct {
	keyword  []byte
	name     []byte
	equals   []byte
	value    *BlockNode
	simi     []byte
	comments *childComments
}

func (n *VarNode) Js() string {
	if len(n.equals) == 0 {
		return n.comments.injectBetween(
			string(n.keyword),
			string(n.name),
			string(n.simi),
		)
	}

	return n.comments.injectBetween(
		string(n.keyword),
		string(n.name),
		string(n.equals),
		n.value.Js(),
		string(n.simi),
	)
}

func (n *VarNode) VarName() string {
	return trimLeftSpaces(n.name)
}

// A FuncNode represents a js function.
type FuncNode struct {
	keyword  []byte
	name     []byte
	params   []byte
	body     *BlockNode
	comments *childComments
}

func (n *FuncNode) Js() string {
	return n.comments.injectBetween(
		string(n.keyword),
		string(n.name),
		string(n.params),
		n.body.Js(),
	)
}

func (n *FuncNode) VarName() (string, bool) {
	if len(n.name) == 0 {
		return "", false
	}

	return trimLeftSpaces(n.name), true
}

// A ClassNode represents a js class.
type ClassNode struct {
	classKeyword   []byte
	name           []byte
	extendsKeyword []byte
	superName      []byte
	body           *BlockNode
	comments       *childComments
}

func (n *ClassNode) Js() string {
	return n.comments.injectBetween(
		string(n.classKeyword),
		string(n.name),
		string(n.extendsKeyword),
		string(n.superName),
		n.body.Js(),
	)
}

func (n *ClassNode) VarName() (string, bool) {
	if len(n.name) == 0 {
		return "", false
	}

	return trimLeftSpaces(n.name), true
}

// An IfNode represents a js if/else[if] statement.
type IfNode struct {
	ifKeyword   []byte
	params      []byte
	ifBody      *BlockNode
	elseKeyword []byte
	elseBody    *BlockNode
	elseIfNode  *IfNode
	comments    *childComments
}

func (n *IfNode) Js() string {
	if len(n.elseKeyword) == 0 {
		return n.comments.injectBetween(
			string(n.ifKeyword),
			string(n.params),
			n.ifBody.Js(),
		)
	}

	if n.elseBody != nil {
		return n.comments.injectBetween(
			string(n.ifKeyword),
			string(n.params),
			n.ifBody.Js(),
			string(n.elseKeyword),
			n.elseBody.Js(),
		)
	}

	js := n.comments.injectBetween(
		string(n.ifKeyword),
		string(n.params),
		n.ifBody.Js(),
		string(n.elseKeyword),
	)
	js += n.elseIfNode.Js()
	return js
}

// A basicCtrlStructNode represents basic js controll structures.
type basicCtrlStructNode struct {
	keyword  []byte
	params   []byte
	body     *BlockNode
	comments *childComments
}

func (n *basicCtrlStructNode) Js() string {
	return n.comments.injectBetween(
		string(n.keyword),
		string(n.params),
		n.body.Js(),
	)
}

// A SwitchNode represents a js switch statement.
type SwitchNode struct {
	basicCtrlStructNode
}

// A WithNode represents a js with statement.
type WithNode struct {
	basicCtrlStructNode
}

// A ForNode represents a js for loop.
type ForLoopNode struct {
	basicCtrlStructNode
}

// A WhileNode represents a js while loop.
type WhileLoopNode struct {
	basicCtrlStructNode
}

// A DoWhileNode represents a js do while loop.
type DoWhileLoopNode struct {
	doKeyword    []byte
	body         *BlockNode
	whileKeyword []byte
	params       []byte
	simi         []byte
	comments     *childComments
}

func (n *DoWhileLoopNode) Js() string {
	return n.comments.injectBetween(
		string(n.doKeyword),
		n.body.Js(),
		string(n.whileKeyword),
		string(n.params),
		string(n.simi),
	)
}

// A TryCatchNode represents a js try catch statement.
type TryCatchNode struct {
	tryKeyword     []byte
	tryBody        *BlockNode
	catchKeyword   []byte
	params         []byte
	catchBody      *BlockNode
	finallyKeyword []byte
	finallyBody    *BlockNode
	comments       *childComments
}

func (n *TryCatchNode) Js() string {
	if len(finallyKeyword) == 0 {
		return n.comments.injectBetween(
			string(n.tryKeyword),
			n.tryBody.Js(),
			string(n.catchKeyword),
			string(n.params),
			n.catchBody.Js(),
		)
	}

	return n.comments.injectBetween(
		string(n.tryKeyword),
		n.tryBody.Js(),
		string(n.catchKeyword),
		string(n.params),
		n.catchBody.Js(),
		string(n.finallyKeyword),
		n.finallyBody.Js(),
	)
}

// A BlockNode represents a block of js code that is not one of the other node types.
type BlockNode struct {
	content []byte
}

func (n *BlockNode) Js() string {
	return string(n.content)
}

func trimLeftSpaces(data []byte) string {
	return strings.TrimLeftFunc(string(data), unicode.IsSpace)
}
