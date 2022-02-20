package js

import "fmt"

// A rewriteFunc will take lexer tokenType/data and return rewritten version of the data.
type rewriteFunc func([]byte) []byte

// rewriteParser will call the rewriteFunc for everything emited by the lexer, and merge the returned data.
func rewriteParser(lex *lexer, rw rewriteFunc) []byte {
	rwData := []byte{}
	for tt, data := lex.Next(); tt != eofType; {
		switch tt {
		case fragmentType, commentType:
			rwData = append(rwData, data...)
		case targetType:
			rwData = append(rwData, rw(data)...)
		default:
			fmt.Println(tt)
			panic("Invalid token type emited from lexFn for rewriteParser")
		}

		tt, data = lex.Next()
	}
	return rwData
}

// lexAssignments will tokenize a javascript block (as output by lexScript) to find assignments.
func lexRewriteAssignments(lastLex lexFn) lexFn {
	var lexRewriteAssignmentsFunc lexFn
	lexRewriteAssignmentsFunc = func(lex *codeLexer) lexFn {
		acceeptAndEmitAssignment := func() bool {
			currPos := lex.nextPos
			if lex.acceptVarName() {
				lex.acceptSpaces()
				if lex.acceptExact(eqOp) && !lex.acceptExact(eqOp) {
					lex.acceptCodeBlock()
					assignPos := lex.nextPos

					lex.nextPos = currPos
					lex.emit(fragmentType)

					lex.nextPos = assignPos
					lex.emit(targetType)
					return true
				}
			}
			lex.nextPos = currPos
			return false
		}

		var skpr skipper
		switch {
		case lex.atEnd():
			lex.emit(fragmentType)
			return lastLex
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
