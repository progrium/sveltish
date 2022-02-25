package js

import (
	"bytes"
)

type varInfo struct {
	index int
	name  string
}

type RewriteInfo []*varInfo

func NewEmptyRewriteInfo() RewriteInfo {
	return RewriteInfo{}
}

func MergeRewriteInfo(allInfo ...RewriteInfo) RewriteInfo {
	infoMap := map[int]string{}
	for _, ri := range allInfo {
		for _, info := range ri {
			if _, exists := infoMap[info.index]; exists {
				continue
			}

			infoMap[info.index] = info.name
		}
	}

	newInfo := NewEmptyRewriteInfo()
	for i, name := range infoMap {
		newInfo = append(newInfo, &varInfo{i, name})
	}
	return newInfo
}

func (info RewriteInfo) VarNames() []string {
	names := []string{}
	for _, v := range info {
		names = append(names, v.name)
	}
	return names
}

func (info RewriteInfo) Dirty() int {
	dirty := 0
	for _, v := range info {
		dirty += 1 << v.index
	}
	return dirty
}

type VarRewriter interface {
	Rewrite([]byte) ([]byte, RewriteInfo)
}

type RewriteFn func(int, string, Var, []byte) []byte

type baseVarRewriter struct {
	vars []Var
	fn   RewriteFn
}

type AssignmentRewriter struct {
	baseVarRewriter
}

func NewAssignmentRewriter(s *Script, fn RewriteFn) *AssignmentRewriter {
	return &AssignmentRewriter{baseVarRewriter{s.rootVars(), fn}}
}

func (rw *AssignmentRewriter) Rewrite(data []byte) ([]byte, RewriteInfo) {
	lex := startNewLexer(lexrewriteAssignments, data)

	info := NewEmptyRewriteInfo()
	newData := rewriteParser(lex, func(currData []byte) []byte {
		i := -1
		for _, v := range rw.vars {
			for _, name := range v.VarNames() {
				i += 1
				if !bytes.HasPrefix(currData, []byte(name)) {
					continue
				}

				info = append(info, &varInfo{i, name})
				return rw.fn(i, name, v, currData)
			}
		}
		return currData
	})

	return newData, info
}

// lexAssignments will tokenize a javascript block (as output by lexScript) to find assignments.
func lexrewriteAssignments(lastLex lexFn) lexFn {
	var lexrewriteAssignmentsFunc lexFn
	lexrewriteAssignmentsFunc = func(lex *codeLexer) lexFn {
		acceeptAndEmitAssignment := func() bool {
			currPos := lex.nextPos
			if !lex.acceptVarName() {
				return false
			}

			lex.acceptSpaces()
			switch {
			case lex.acceptExact(plusEqOp):
				break
			case lex.acceptExact(minusEqOp):
				break
			case lex.acceptExact(eqOp) && !lex.acceptExact(eqOp):
				break
			default:
				lex.nextPos = currPos
				return false
			}

			lex.acceptCodeBlock()
			assignPos := lex.nextPos

			lex.nextPos = currPos
			lex.emit(fragmentType)

			lex.nextPos = assignPos
			lex.emit(targetType)
			return true

		}

		var skpr skipper
		switch {
		case lex.atEnd():
			lex.emit(fragmentType)
			return lastLex
		case lex.acceptExact(dotOp), lex.acceptExact(optnlDotOp):
			lex.acceptSpaces()
			lex.acceptVarName()
			return lexrewriteAssignmentsFunc
		case lex.acceptExact(lineCommentOpen):
			skpr = newLineCommentSkipper()
		case lex.acceptExact(blockCommentOpen):
			skpr = newBlockCommentSkipper()
		case lex.acceptExact(singleQuote):
			skpr = newSingleQuoteSkipper()
		case lex.acceptExact(doubleQuote):
			skpr = newDoubleQuoteSkipper()
		case lex.acceptExact(tmplQuote):
			skpr = newTmplQuoteSkipper()
		case lex.acceptExact(regexQuote):
			skpr = newRegexQuoteSkipper()
		default:
			if !acceeptAndEmitAssignment() {
				lex.pop()
			}
			return lexrewriteAssignmentsFunc
		}

		lex.skip(skpr, func(_ byte) {
			switch open, _ := skpr.group(); string(open) {
			case lineCommentOpen, blockCommentOpen, singleQuote, doubleQuote, tmplQuote, regexQuote:
				return
			}

			acceeptAndEmitAssignment()
		})
		return lexrewriteAssignmentsFunc
	}
	return lexrewriteAssignmentsFunc
}

type VarNameRewriter struct {
	baseVarRewriter
}

func NewVarNameRewriter(s *Script, fn RewriteFn) *VarNameRewriter {
	return &VarNameRewriter{baseVarRewriter{s.rootVars(), fn}}
}

func (rw *VarNameRewriter) Rewrite(data []byte) ([]byte, RewriteInfo) {
	lex := startNewLexer(lexRewriteVarNames, data)

	info := NewEmptyRewriteInfo()
	newData := rewriteParser(lex, func(currData []byte) []byte {
		i := -1
		for _, v := range rw.vars {
			for _, name := range v.VarNames() {
				i += 1
				if bytes.Compare(currData, []byte(name)) != 0 {
					continue
				}

				info = append(info, &varInfo{i, name})
				return rw.fn(i, name, v, currData)
			}
		}
		return currData
	})

	return newData, info
}

// lexRewriteVarNames will tokenize a javascript block (as output by lexScript) to find variable names (and keywords).
func lexRewriteVarNames(lastLex lexFn) lexFn {
	var lexRewriteVarNamesFunc lexFn
	lexRewriteVarNamesFunc = func(lex *codeLexer) lexFn {
		acceeptAndEmitVarName := func() bool {
			currPos := lex.nextPos
			if !lex.acceptVarName() {
				return false
			}
			varPos := lex.nextPos

			lex.nextPos = currPos
			lex.emit(fragmentType)

			lex.nextPos = varPos
			lex.emit(targetType)
			return true
		}

		switch {
		case lex.atEnd():
			lex.emit(fragmentType)
			return lastLex
		case lex.acceptExact(dotOp), lex.acceptExact(optnlDotOp):
			lex.acceptSpaces()
			lex.acceptVarName()
			return lexRewriteVarNamesFunc
		case lex.acceptExact(lineCommentOpen):
			lex.skip(newLineCommentSkipper(), nil)
			return lexRewriteVarNamesFunc
		case lex.acceptExact(blockCommentOpen):
			lex.skip(newBlockCommentSkipper(), nil)
			return lexRewriteVarNamesFunc
		case lex.acceptExact(singleQuote):
			lex.skip(newSingleQuoteSkipper(), nil)
			return lexRewriteVarNamesFunc
		case lex.acceptExact(doubleQuote):
			lex.skip(newDoubleQuoteSkipper(), nil)
			return lexRewriteVarNamesFunc
		case lex.acceptExact(tmplQuote):
			skpr := newTmplQuoteSkipper()
			lex.skip(skpr, func(_ byte) {
				switch open, _ := skpr.group(); string(open) {
				case parenOpen, curlyOpen, tmplQuoteExprOpen:
					acceeptAndEmitVarName()
				}
			})
			return lexRewriteVarNamesFunc
		case lex.acceptExact(regexQuote):
			lex.skip(newRegexQuoteSkipper(), nil)
			return lexRewriteVarNamesFunc
		}

		if !acceeptAndEmitVarName() {
			lex.pop()
		}
		return lexRewriteVarNamesFunc
	}
	return lexRewriteVarNamesFunc
}

// rewriteParser will call the rewriteFunc for everything emited by the lexer, and merge the returned data.
func rewriteParser(lex *lexer, rw func([]byte) []byte) []byte {
	rwData := [][]byte{}
	for tt, data := lex.Next(); tt != eofType; {
		switch tt {
		case fragmentType, commentType:
			rwData = append(rwData, data)
		case targetType:
			rwData = append(rwData, rw(data))
		default:
			panic("Invalid token type emited from lexFn for rewriteParser")
		}

		tt, data = lex.Next()
	}
	return bytes.Join(rwData, nil)
}

type rewriteAssignmenter interface {
	rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo)
}

func (n *Script) RewriteForInstance(
	rw VarRewriter,
	wrapUpd func(RewriteInfo, []byte) []byte,
	wrapUpds func([]byte) []byte,
) ([]byte, RewriteInfo) {
	nrmlRoots := []Node{}
	ratvRoots := []*LabelNode{}
	for _, n := range n.roots {
		if ln, ok := n.(*LabelNode); ok && ln.IsReactive() {
			ratvRoots = append(ratvRoots, ln)
			continue
		}

		nrmlRoots = append(nrmlRoots, n)
	}

	info := NewEmptyRewriteInfo()
	i := 0
	for _, v := range n.rootVars() {
		for _, name := range v.VarNames() {
			info = append(info, &varInfo{i, name})
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

	updsData := [][]byte{}
	for _, r := range ratvRoots {
		updData, updInfo := r.rewriteAssignments(rw)
		updsData = append(updsData, wrapUpd(updInfo, updData))
	}
	data = append(data, wrapUpds(bytes.Join(updsData, nil)))

	return bytes.Join(data, nil), info
}

func (n *Script) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data := [][]byte{}
	info := []RewriteInfo{}
	for _, n := range n.roots {
		if ra, ok := n.(rewriteAssignmenter); ok {
			rootData, rootInfo := ra.rewriteAssignments(rw)
			data = append(data, rootData)
			info = append(info, rootInfo)
		} else {
			data = append(data, []byte(n.Js()))
		}
	}
	return bytes.Join(data, nil), MergeRewriteInfo(info...)
}

func (n *LabelNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data := [][]byte{}
	info := []RewriteInfo{}

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
	return n.comments.injectBetween(data...), MergeRewriteInfo(info...)
}

func (n *VarNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
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
	return n.comments.injectBetween(data...), NewEmptyRewriteInfo()
}

func (n *FuncNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.keyword,
		n.name,
		n.params,
		data,
	), info
}

func (n *ClassNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.classKeyword,
		n.name,
		n.extendsKeyword,
		n.superName,
		data,
	), info
}

func (n *IfNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data := [][]byte{}
	info := []RewriteInfo{}

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

	return n.comments.injectBetween(data...), MergeRewriteInfo(info...)
}

func (n *basicCtrlStructNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data, info := n.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.keyword,
		n.params,
		data,
	), info
}

func (n *DoWhileLoopNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data, info := n.body.rewriteAssignments(rw)
	return n.comments.injectBetween(
		n.doKeyword,
		data,
		n.whileKeyword,
		n.params,
		n.simi,
	), info
}

func (n *TryCatchNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	data := [][]byte{}

	data = append(data, n.tryKeyword)

	tryData, tryInfo := n.tryBody.rewriteAssignments(rw)
	data = append(data, tryData)
	data = append(data, n.catchKeyword)
	data = append(data, n.params)

	catchData, catchInfo := n.catchBody.rewriteAssignments(rw)
	data = append(data, catchData)

	if n.finallyBody == nil {
		return n.comments.injectBetween(data...), MergeRewriteInfo(tryInfo, catchInfo)
	}

	data = append(data, n.finallyKeyword)

	finallyData, finallyInfo := n.finallyBody.rewriteAssignments(rw)
	data = append(data, finallyData)

	return n.comments.injectBetween(data...), MergeRewriteInfo(tryInfo, catchInfo, finallyInfo)
}

func (n *BlockNode) rewriteAssignments(rw VarRewriter) ([]byte, RewriteInfo) {
	return rw.Rewrite(n.content)
}
