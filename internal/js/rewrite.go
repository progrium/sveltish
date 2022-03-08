package js

import (
	"bytes"
	"unicode"
)

type VarRewriter interface {
	Rewrite([]byte) ([]byte, *VarsInfo)
}

type VarsInfo struct {
	indexes []int
	names   []string
}

func NewEmptyVarsInfo() *VarsInfo {
	return &VarsInfo{}
}

func MergeVarsInfo(allInfo ...*VarsInfo) *VarsInfo {
	newInfo := NewEmptyVarsInfo()
	for _, info := range allInfo {
		for i, varIndex := range info.indexes {
			varName := info.names[i]
			newInfo.insert(varIndex, varName)
		}
	}
	return newInfo
}

func (info *VarsInfo) Names() []string {
	return info.names
}

func (info *VarsInfo) Dirty() int {
	dirty := 0
	for _, varIndex := range info.indexes {
		dirty += 1 << varIndex
	}
	return dirty
}

func (info *VarsInfo) insert(newVarIndex int, newVarName string) {
	for i, varIndex := range info.indexes {
		if newVarIndex == varIndex {
			if info.names[i] != newVarName {
				panic("Trying to add multiple vars at the same index")
			}
			return
		}
	}
	info.indexes = append(info.indexes, newVarIndex)
	info.names = append(info.names, newVarName)
}

type RewriteFn func(int, string, Var, []byte) []byte

type lexVarRewriter struct {
	vars    []Var
	fn      RewriteFn
	lexInit func(lexFn) lexFn
	hasVar  func([]byte, []byte) bool
}

func NewAssignmentRewriter(s *Script, fn RewriteFn) VarRewriter {
	rootVars := []Var{}
	if s != nil {
		rootVars = s.rootVars()
	}

	return &lexVarRewriter{
		rootVars,
		fn,
		lexRewriteAssignments,
		func(data, name []byte) bool {
			if !bytes.HasPrefix(data, name) {
				return false
			}
			if len(data) == len(name) {
				return true
			}

			rest := data[len(name):]
			switch {
			case unicode.IsSpace(rune(rest[0])):
				return true
			case bytes.HasPrefix(rest, []byte(eqOp)):
				return true
			case bytes.HasPrefix(rest, []byte(plusEqOp)):
				return true
			case bytes.HasPrefix(rest, []byte(minusEqOp)):
				return true
			}

			return false
		},
	}
}

func NewVarNameRewriter(s *Script, fn RewriteFn) VarRewriter {
	rootVars := []Var{}
	if s != nil {
		rootVars = s.rootVars()
	}

	return &lexVarRewriter{
		rootVars,
		fn,
		lexRewriteVarNames,
		func(data, name []byte) bool {
			return bytes.Compare(data, name) == 0
		},
	}
}

func (rw *lexVarRewriter) Rewrite(data []byte) ([]byte, *VarsInfo) {
	lex := startNewLexer(rw.lexInit, data)
	info := NewEmptyVarsInfo()
	newData := rewriteParser(lex, func(currData []byte) []byte {
		i := -1
		for _, v := range rw.vars {
			for _, name := range v.VarNames() {
				i += 1
				if !rw.hasVar(currData, []byte(name)) {
					continue
				}

				info.insert(i, name)

				if rw.fn == nil {
					return currData
				}
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
					lex.backup()
					if !acceeptAndEmitVarName() {
						lex.pop()
					}
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

// rewriteParser will call the rewriteFunc for every rewrite target emited by the lexer, and merge the returned data.
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
