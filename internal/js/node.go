package js

import (
	"bytes"
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

type rewriteAssignmenter interface {
	rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo)
}

type WrapUpdFn func(*VarsInfo, []byte) []byte
type WrapUpdsFn func(func(WrapUpdFn) []byte) []byte

// RewriteForInstance creates the js for the svelet runtime instance function
func (n *Script) RewriteForInstance(
	rw VarRewriter,
	wrapUpds WrapUpdsFn,
) ([]byte, *VarsInfo) {
	nrmlRoots := []Node{}
	ratvRoots := []*LabelNode{}
	for _, n := range n.roots {
		if ln, ok := n.(*LabelNode); ok && ln.IsReactive() {
			ratvRoots = append(ratvRoots, ln)
			continue
		}

		nrmlRoots = append(nrmlRoots, n)
	}

	info := NewEmptyVarsInfo()
	i := 0
	for _, v := range n.rootVars() {
		for _, name := range v.VarNames() {
			info.insert(i, name)
			i += 1
		}
	}

	data := [][]byte{}
	for _, r := range ratvRoots {
		if len(r.name) == 0 {
			continue
		}

		data = append(data, []byte("\nlet "+string(r.name)+";"))
	}

	for _, r := range n.roots {
		if n, ok := r.(rewriteAssignmenter); ok {
			nData, _ := n.rewriteAssignments(rw)
			data = append(data, nData)
		} else {
			data = append(data, []byte(r.Js()))
		}
	}
	if len(ratvRoots) == 0 {
		return bytes.Join(data, nil), info
	}

	data = append(
		data,
		wrapUpds(func(wrapUpd WrapUpdFn) []byte {
			updsData := [][]byte{}
			for _, r := range ratvRoots {
				updData, updInfo := r.rewriteAssignments(rw)
				updsData = append(
					updsData,
					wrapUpd(updInfo, updData),
				)
			}
			return bytes.Join(updsData, nil)
		}),
	)

	return bytes.Join(data, nil), info
}

func (n *Script) Js() string {
	return noRewriteJs(n)
}

func (n *Script) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data := [][]byte{}
	info := []*VarsInfo{}
	for _, n := range n.roots {
		if ra, ok := n.(rewriteAssignmenter); ok {
			rootData, rootInfo := ra.rewriteAssignments(rw)
			data = append(data, rootData)
			info = append(info, rootInfo)
		} else {
			data = append(data, []byte(n.Js()))
		}
	}
	return bytes.Join(data, nil), MergeVarsInfo(info...)
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

func (n *LabelNode) Js() string {
	return noRewriteJs(n)
}

func (n *LabelNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data := [][]byte{}
	info := []*VarsInfo{}

	data = append(data, n.label)

	if len(n.name) != 0 {
		data = append(data, n.name)
		data = append(data, n.equals)
	}

	if ra, ok := n.body.(rewriteAssignmenter); ok {
		raData, raInfo := ra.rewriteAssignments(rw)
		data = append(data, raData)
		info = append(info, raInfo)
	} else {
		data = append(data, []byte(n.body.Js()))
	}

	if len(n.name) != 0 {
		data = append(data, n.simi)
	}

	//TODO need to add prefix and sufix
	return n.comments.injectBetween(data...), MergeVarsInfo(info...)
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

func (n *VarNode) Js() string {
	return noRewriteJs(n)
}

func (n *VarNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data := [][]byte{}

	data = append(data, n.keyword)
	data = append(data, n.name)

	if len(n.equals) != 0 {
		data = append(data, n.equals)

		valueData, info := n.value.rewriteAssignments(rw)
		data = append(data, valueData)
		data = append(data, n.simi)
		return n.comments.injectBetween(data...), info
	}

	data = append(data, n.simi)
	return n.comments.injectBetween(data...), NewEmptyVarsInfo()
}

// A FuncNode represents a js function.
type FuncNode struct {
	keyword  []byte
	name     []byte
	params   []byte
	body     *BlockNode
	comments *childComments
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

func (n *FuncNode) Js() string {
	return noRewriteJs(n)
}

func (n *FuncNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.keyword,
		n.name,
		n.params,
		data,
	), info
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

func (n *ClassNode) VarNames() []string {
	if len(n.name) == 0 {
		return nil
	}

	return []string{trimLeftSpaces(n.name)}
}

func (n *ClassNode) Js() string {
	return noRewriteJs(n)
}

func (n *ClassNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.classKeyword,
		n.name,
		n.extendsKeyword,
		n.superName,
		data,
	), info
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

func (n *IfNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data := [][]byte{}
	info := []*VarsInfo{}

	data = append(data, n.ifKeyword)
	data = append(data, n.params)

	ifData, ifInfo := n.ifBody.rewriteAssignments(rw)
	data = append(data, ifData)
	info = append(info, ifInfo)

	if n.elseBody != nil {
		data = append(data, n.elseKeyword)

		elseData, elseInfo := n.elseBody.rewriteAssignments(rw)
		data = append(data, elseData)
		info = append(info, elseInfo)
	}

	if n.elseIfNode != nil {
		data = append(data, n.elseKeyword)

		elseIfData, elseIfInfo := n.elseIfNode.rewriteAssignments(rw)
		data = append(data, elseIfData)
		info = append(info, elseIfInfo)
	}

	return n.comments.injectBetween(data...), MergeVarsInfo(info...)
}

// A basicCtrlStructNode represents basic js controll structures.
type basicCtrlStructNode struct {
	keyword  []byte
	params   []byte
	body     *BlockNode
	comments *childComments
}

func (n *basicCtrlStructNode) Js() string {
	return noRewriteJs(n)
}

func (n *basicCtrlStructNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.keyword,
		n.params,
		data,
	), info
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
	return noRewriteJs(n)
}

func (n *DoWhileLoopNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.doKeyword,
		data,
		n.whileKeyword,
		n.params,
		n.simi,
	), info
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
	return noRewriteJs(n)
}

func (n *TryCatchNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	data := [][]byte{}

	data = append(data, n.tryKeyword)

	tryData, tryInfo := n.tryBody.rewriteAssignments(rw)
	data = append(data, tryData)
	data = append(data, n.catchKeyword)
	data = append(data, n.params)

	catchData, catchInfo := n.catchBody.rewriteAssignments(rw)
	data = append(data, catchData)

	if n.finallyBody == nil {
		return n.comments.injectBetween(data...), MergeVarsInfo(tryInfo, catchInfo)
	}

	data = append(data, n.finallyKeyword)

	finallyData, finallyInfo := n.finallyBody.rewriteAssignments(rw)
	data = append(data, finallyData)

	return n.comments.injectBetween(data...), MergeVarsInfo(tryInfo, catchInfo, finallyInfo)
}

// A BlockNode represents a block of js code that is not one of the other node types.
type BlockNode struct {
	content []byte
}

func (n *BlockNode) Js() string {
	return noRewriteJs(n)
}

func (n *BlockNode) rewriteAssignments(rw VarRewriter) ([]byte, *VarsInfo) {
	if rw == nil {
		return n.content, NewEmptyVarsInfo()
	}

	return rw.Rewrite(n.content)
}

func noRewriteJs(rw rewriteAssignmenter) string {
	data, _ := rw.rewriteAssignments(nil)
	return string(data)
}

func trimLeftSpaces(data []byte) string {
	return strings.TrimLeftFunc(string(data), unicode.IsSpace)
}
