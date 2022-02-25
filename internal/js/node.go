package js

import (
	"strings"
	"unicode"
)

// All node types implement the Node interface.
type Node interface {
	Js() string
}

// All node types that set a variable name implement the Var interface.
type Var interface {
	Node
	VarType() string
	VarNames() []string
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

func (n *Script) rootVars() []Var {
	vars := []Var{}
	for _, r := range n.roots {
		v, ok := r.(Var)
		if !ok {
			continue
		}

		vars = append(vars, v)
	}
	return vars
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

func (cs *childComments) injectBetween(allData ...[]byte) []byte {
	js := []byte{}
	notSet := 0
	for i, data := range allData {
		if len(data) == 0 {
			notSet += 1
			continue
		}

		js = append(js, data...)

		for len(cs.nodes) <= i-notSet {
			cs.appendNil()
		}
		if n := cs.nodes[i-notSet]; n != nil {
			js = append(js, []byte(n.Js())...)
		}
	}
	return js
}

// A LabelNode reprents a labeled js statement.
type LabelNode struct {
	label    []byte
	name     []byte
	equals   []byte
	body     Node
	simi     []byte
	comments *childComments
}

func (n *LabelNode) Js() string {
	if len(n.name) == 0 {
		return string(n.comments.injectBetween(
			n.label,
			[]byte(n.body.Js()),
		))
	}

	return string(n.comments.injectBetween(
		n.label,
		n.name,
		n.equals,
		[]byte(n.body.Js()),
		n.simi,
	))
}

func (n *LabelNode) VarType() string {
	return trimLeftSpaces(n.label)
}

func (n *LabelNode) VarNames() []string {
	if len(n.name) == 0 {
		return nil
	}

	//TODO, add support for destructuring
	return []string{trimLeftSpaces(n.name)}
}

func (n *LabelNode) Label() string {
	return strings.TrimFunc(string(n.label), func(r rune) bool {
		switch {
		case unicode.IsSpace(r):
			return true
		case r == rune(labelSufix[0]):
			return true
		}

		return false
	})
}

const reactiveLabel = "$"

func (n *LabelNode) IsReactive() bool {
	return n.Label() == reactiveLabel
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
		return string(n.comments.injectBetween(
			n.keyword,
			n.name,
			n.simi,
		))
	}

	return string(n.comments.injectBetween(
		n.keyword,
		n.name,
		n.equals,
		[]byte(n.value.Js()),
		n.simi,
	))
}

func (n *VarNode) VarType() string {
	return trimLeftSpaces(n.keyword)
}

func (n *VarNode) VarNames() []string {
	if len(n.name) == 0 {
		return nil
	}

	//TODO, add support for destructuring
	return []string{trimLeftSpaces(n.name)}
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
	return string(n.comments.injectBetween(
		n.keyword,
		n.name,
		n.params,
		[]byte(n.body.Js()),
	))
}

func (n *FuncNode) VarType() string {
	return funcKeyword
}

func (n *FuncNode) VarNames() []string {
	if len(n.name) == 0 {
		return nil
	}

	return []string{trimLeftSpaces(n.name)}
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
	return string(n.comments.injectBetween(
		n.classKeyword,
		n.name,
		n.extendsKeyword,
		n.superName,
		[]byte(n.body.Js()),
	))
}

func (n *ClassNode) VarNames() []string {
	if len(n.name) == 0 {
		return nil
	}

	return []string{trimLeftSpaces(n.name)}
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
		return string(n.comments.injectBetween(
			n.ifKeyword,
			n.params,
			[]byte(n.ifBody.Js()),
		))
	}

	if n.elseBody != nil {
		return string(n.comments.injectBetween(
			n.ifKeyword,
			n.params,
			[]byte(n.ifBody.Js()),
			n.elseKeyword,
			[]byte(n.elseBody.Js()),
		))
	}

	js := string(n.comments.injectBetween(
		n.ifKeyword,
		n.params,
		[]byte(n.ifBody.Js()),
		n.elseKeyword,
	))
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
	return string(n.comments.injectBetween(
		n.keyword,
		n.params,
		[]byte(n.body.Js()),
	))
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
	return string(n.comments.injectBetween(
		n.doKeyword,
		[]byte(n.body.Js()),
		n.whileKeyword,
		n.params,
		n.simi,
	))
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
		return string(n.comments.injectBetween(
			n.tryKeyword,
			[]byte(n.tryBody.Js()),
			n.catchKeyword,
			n.params,
			[]byte(n.catchBody.Js()),
		))
	}

	return string(n.comments.injectBetween(
		n.tryKeyword,
		[]byte(n.tryBody.Js()),
		n.catchKeyword,
		n.params,
		[]byte(n.catchBody.Js()),
		n.finallyKeyword,
		[]byte(n.finallyBody.Js()),
	))
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
