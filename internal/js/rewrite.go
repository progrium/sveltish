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

type lexVarRewriter struct {
	vars    []Var
	fn      RewriteFn
	lexInit func(lexFn) lexFn
	hasVar  func([]byte, []byte) bool
}

func NewAssignmentRewriter(s *Script, fn RewriteFn) *lexVarRewriter {
	return &lexVarRewriter{
		s.rootVars(),
		fn,
		lexRewriteAssignments,
		func(data, name []byte) bool {
			return bytes.HasPrefix(data, []byte(name))
		},
	}
}

func NewVarNameRewriter(s *Script, fn RewriteFn) *lexVarRewriter {
	return &lexVarRewriter{
		s.rootVars(),
		fn,
		lexRewriteVarNames,
		func(data, name []byte) bool {
			return bytes.Compare(data, []byte(name)) != 0
		},
	}
}

func (rw *lexVarRewriter) Rewrite(data []byte) ([]byte, RewriteInfo) {
	lex := startNewLexer(rw.lexInit, data)
	info := NewEmptyRewriteInfo()
	newData := rewriteParser(lex, func(currData []byte) []byte {
		i := -1
		for _, v := range rw.vars {
			for _, name := range v.VarNames() {
				i += 1
				if !rw.hasVar(currData, []byte(name)) {
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
func lexRewriteAssignments(lastLex lexFn) lexFn {
	var lexRewriteAssignmentsFunc lexFn
	lexRewriteAssignmentsFunc = func(lex *codeLexer) lexFn {
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
			return lexRewriteAssignmentsFunc
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
			return lexRewriteAssignmentsFunc
		}

		lex.skip(skpr, func(_ byte) {
			switch open, _ := skpr.group(); string(open) {
			case lineCommentOpen, blockCommentOpen, singleQuote, doubleQuote, tmplQuote, regexQuote:
				return
			}

			acceeptAndEmitAssignment()
		})
		return lexRewriteAssignmentsFunc
	}
	return lexRewriteAssignmentsFunc
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
