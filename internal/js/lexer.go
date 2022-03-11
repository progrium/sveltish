package js

import (
	"bytes"
	"errors"
	"unicode"
)

const (
	lineCommentOpen    = "//"
	blockCommentOpen   = "/*"
	blockCommentClose  = "*/"
	parenOpen          = "("
	parenClose         = ")"
	curlyOpen          = "{"
	curlyClose         = "}"
	quoteEscape        = `\`
	singleQuote        = "'"
	doubleQuote        = `"`
	tmplQuote          = "`"
	tmplQuoteExprOpen  = "${"
	tmplQuoteExprClose = "}"
	regexQuote         = "/"
	labelSufix         = ":"
	varKeyword         = "var"
	letKeyword         = "let"
	constKeyword       = "const"
	funcKeyword        = "function"
	ifKeyword          = "if"
	elseKeyword        = "else"
	forKeyword         = "for"
	whileKeyword       = "while"
	doKeyword          = "do"
	switchKeyword      = "switch"
	withKeyword        = "with"
	tryKeyword         = "try"
	catchKeyword       = "catch"
	finallyKeyword     = "finally"
	classKeyword       = "class"
	extendsKeyword     = "extends"
	arrowFuncOp        = "=>"
	eqOp               = "="
	plusEqOp           = "+="
	minusEqOp          = "-="
	simiOp             = ";"
	dotOp              = "."
	optnlDotOp         = "?."
	newLine            = "\n"
)

// tokenType identifies the type of lex items.
type tokenType int

const (
	// shared types
	eofType   tokenType = -1
	errorType tokenType = iota
	commentType

	// lexScript types
	keywordType
	varNameType
	labelType
	eqOpType
	simiOpType
	paramsType
	codeBlockType

	// rewrite types
	targetType
	fragmentType
)

func (tt tokenType) String() string {
	switch tt {
	case eofType:
		return "eof"
	case errorType:
		return "error"
	case commentType:
		return "comment"
	case keywordType:
		return "keyword"
	case varNameType:
		return "varName"
	case labelType:
		return "label"
	case eqOpType:
		return "eqOp"
	case simiOpType:
		return "simiOp"
	case paramsType:
		return "params"
	case codeBlockType:
		return "codeBlock"
	case targetType:
		return "target"
	case fragmentType:
		return "fragment"
	}

	return "Unkown token type"
}

type lexerItem struct {
	tt   tokenType
	data []byte
}

// lexer is the api for the codeLexer.
type lexer struct {
	lex   *codeLexer
	stack []lexerItem
	err   error
}

// startNewLexer creates and starts a new lexer.
func startNewLexer(initLex func(lexFn) lexFn, data []byte) *lexer {
	lex := newCodeLexer(data)
	go lex.run(initLex)

	return &lexer{
		lex:   lex,
		stack: []lexerItem{},
		err:   nil,
	}
}

// Next returns the next token from the lexer.
func (lex *lexer) Next() (tokenType, []byte) {
	if stackSize := len(lex.stack); stackSize != 0 {
		item := lex.stack[stackSize-1]
		lex.stack = lex.stack[:stackSize-1]

		return item.tt, item.data
	}

	item, ok := <-lex.lex.items
	if !ok {
		return eofType, nil
	}
	if item.tt == errorType {
		lex.err = errors.New(string(item.data))
	}

	return item.tt, item.data
}

// Err returns the error returned from the lexer.
func (lex *lexer) Err() error {
	return lex.err
}

// rewind will re add the given output back to the lexer.
func (lex *lexer) rewind(tt tokenType, data []byte) {
	lex.stack = append(lex.stack, lexerItem{tt, data})
}

// codeLexer holds the state of the scanner.
type codeLexer struct {
	data     []byte
	startPos int
	nextPos  int
	items    chan lexerItem
}

func newCodeLexer(data []byte) *codeLexer {
	return &codeLexer{
		data:     data,
		startPos: 0,
		nextPos:  0,
		items:    make(chan lexerItem),
	}
}

// run starts the lexers output (expected to be in its own goroutine)
func (lex *codeLexer) run(initLex func(lexFn) lexFn) {
	lex.acceptSpaces()
	if lex.acceptComment() {
		lex.emit(commentType)
		lex.acceptSpaces()
	}

	for fn := initLex(nil); fn != nil; {
		fn = fn(lex)
	}
	close(lex.items)
}

// emit passes the item for next to return.
func (lex *codeLexer) emit(tt tokenType) {
	lex.items <- lexerItem{
		tt:   tt,
		data: lex.data[lex.startPos:lex.nextPos],
	}
	lex.startPos = lex.nextPos

	lex.acceptSpaces()
	if lex.acceptComment() {
		lex.emit(commentType)
		lex.acceptSpaces()
	}
}

// emitError passes an error for next to return.
func (lex *codeLexer) emitError(message string) {
	lex.items <- lexerItem{
		tt:   errorType,
		data: []byte(message),
	}
}

// pop will get the next byte from data.
func (lex *codeLexer) pop() (byte, bool) {
	if lex.atEnd() {
		return 0, false
	}

	c := lex.data[lex.nextPos]
	lex.movePos(1)
	return c, true
}

// peek will get but not consume the next byte from data.
func (lex *codeLexer) peek() (byte, bool) {
	c, ok := lex.pop()
	if !ok {
		return 0, false
	}
	lex.backup()

	return c, true
}

// backup go back one byte in the data.
func (lex *codeLexer) backup() {
	lex.movePos(-1)
}

// atEnd check if lexer is at the end of the data
func (lex *codeLexer) atEnd() bool {
	return lex.nextPos == len(lex.data)
}

// movePos will move the lexer by the given value
func (lex *codeLexer) movePos(by int) {
	if lex.nextPos+by < lex.startPos {
		panic("Trying to move currPos before start data")
	}
	if lex.nextPos+by > len(lex.data) {
		panic("Trying to move currPos outside of data")
	}

	lex.nextPos += by
}

// acceptSpaces will add the string of spaces to the current lex token.
func (lex *codeLexer) acceptSpaces() bool {
	c, ok := lex.pop()
	if !ok {
		return false
	}

	foundSpace := false
	for unicode.IsSpace(rune(c)) {
		foundSpace = true

		c, ok = lex.pop()
		if !ok {
			return true
		}
	}

	lex.backup()
	return foundSpace
}

// acceptComment will add a valid comment to the current lex token
func (lex *codeLexer) acceptComment() bool {
	switch {
	case lex.acceptExact(lineCommentOpen):
		lex.skip(newLineCommentSkipper(), nil)
		return true
	case lex.acceptExact(blockCommentOpen):
		lex.skip(newBlockCommentSkipper(), nil)
		return true
	}

	return false
}

// acceptExact will add the exact val to the current lex token, if it is next.
func (lex *codeLexer) acceptExact(val string) bool {
	if val == "" {
		return false
	}

	valBytes := []byte(val)
	if found := bytes.HasPrefix(lex.data[lex.nextPos:], valBytes); !found {
		return false
	}

	lex.movePos(len(valBytes))
	return true
}

// acceptVarName will add the given keyword to the current lex token
func (lex *codeLexer) acceptKeyword(kw string) bool {
	if found := lex.acceptExact(kw); !found {
		return false
	}

	next, ok := lex.peek()
	switch {
	case !ok:
		return true
	case unicode.IsSpace(rune(next)):
		return true
	case string(next) == parenOpen:
		return true
	case string(next) == curlyOpen:
		return true
	}

	return false
}

const (
	vaildFirstVarChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_$"
	validVarChars      = vaildFirstVarChars + "0123456789"
)

// acceptVarName will add a valid variable name to the current lex token
func (lex *codeLexer) acceptVarName() bool {
	c, ok := lex.pop()
	if !ok {
		return false
	}

	if !bytes.ContainsAny([]byte{c}, vaildFirstVarChars) {
		lex.backup()
		return false
	}
	for {
		c, ok = lex.pop()
		if !ok {
			return true
		}

		if !bytes.ContainsAny([]byte{c}, validVarChars) {
			lex.backup()
			return true
		}
	}
}

// acceptLabel will add a vaild label to the current lex token
func (lex *codeLexer) acceptLabel() bool {
	currPos := lex.nextPos
	if !lex.acceptVarName() {
		return false
	}

	lex.acceptSpaces()
	if !lex.acceptExact(labelSufix) {
		lex.nextPos = currPos
		return false
	}

	return true
}

// acceptCodeBlock will add a everything until an expr end to the current lex token, i.e. it always return true
func (lex *codeLexer) acceptCodeBlock() bool {
	for {
		switch {
		case lex.acceptEndOfExpr():
			return true
		case lex.acceptExact(curlyOpen):
			lex.skip(newCurlyGroupSkipper(), nil)
		case lex.acceptExact(parenOpen):
			lex.skip(newParenGroupSkipper(), nil)
		case lex.acceptExact(singleQuote):
			lex.skip(newSingleQuoteSkipper(), nil)
		case lex.acceptExact(doubleQuote):
			lex.skip(newDoubleQuoteSkipper(), nil)
		case lex.acceptExact(tmplQuote):
			lex.skip(newTmplQuoteSkipper(), nil)
		case lex.acceptExact(regexQuote):
			lex.skip(newRegexQuoteSkipper(), nil)
		default:
			lex.pop()
		}
	}
}

// acceptEndOfExpr will add a valid expr end to the current lex token, simiOp will not be accepted (just checked for)
func (lex *codeLexer) acceptEndOfExpr() bool {
	c, ok := lex.peek()
	if !ok {
		return true
	}
	if c == byte(simiOp[0]) {
		return true
	}
	//TODO, add auto-semicolon support

	return false
}

// skip will use a skipper to ignore code that does not need lexed
func (lex *codeLexer) skip(skpr skipper, onEach func(byte)) {
	for skpr.isOpen() {
		c, ok := lex.pop()
		if !ok {
			return
		}

		if onEach != nil {
			onEach(c)
		}

		skpr.next(c)
	}
}

// The lexerFuncs are used to lex a sepific part of the js and returns
// another lexerFunc that can lex the next part
type lexFn func(*codeLexer) lexFn

// lexScript will tokenize the root javascript scope
func lexScript(lastLex lexFn) lexFn {
	var lexScriptFn lexFn
	lexScriptFn = func(lex *codeLexer) lexFn {
		switch {
		case lex.atEnd():
			return lastLex
		case lex.acceptExact(simiOp):
			lex.emit(simiOpType)
			return lexScriptFn
		case lex.acceptLabel():
			lex.emit(labelType)
			return lexLabel(lexScriptFn)
		case lex.acceptKeyword(varKeyword), lex.acceptKeyword(letKeyword), lex.acceptKeyword(constKeyword):
			lex.emit(keywordType)
			return lexVar(lexScriptFn)
		case lex.acceptKeyword(funcKeyword):
			lex.emit(keywordType)
			return lexFunction(lexScriptFn)
		case lex.acceptKeyword(ifKeyword):
			lex.emit(keywordType)
			return lexIfStmt(lexScriptFn)
		case lex.acceptKeyword(forKeyword), lex.acceptKeyword(whileKeyword), lex.acceptKeyword(switchKeyword), lex.acceptKeyword(switchKeyword), lex.acceptKeyword(withKeyword):
			lex.emit(keywordType)
			return lexCtrlStruct(lexScriptFn)
		case lex.acceptExact(doKeyword):
			lex.emit(keywordType)
			return lexDoWhile(lexScriptFn)
		case lex.acceptExact(tryKeyword):
			lex.emit(keywordType)
			return lexTryCatch(lexScriptFn)
		case lex.acceptExact(classKeyword):
			lex.emit(keywordType)
			return lexClass(lexScriptFn)
		}

		lex.acceptCodeBlock()
		lex.emit(codeBlockType)
		return lexScriptFn
	}
	return lexScriptFn
}

// lexVar will tokenize a javascript variable defintion, starting after the keyword
func lexVar(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		//TODO allow destructuring
		if found := lex.acceptVarName(); !found {
			lex.emitError("No variable name found")
			return nil
		}
		lex.emit(varNameType)

		if found := lex.acceptExact(eqOp); !found {
			if found := lex.acceptExact(simiOp); found {
				lex.emit(simiOpType)
			}

			return lastLex
		}
		lex.emit(eqOpType)

		lex.acceptCodeBlock()
		lex.emit(codeBlockType)
		if lex.acceptExact(simiOp) {
			lex.emit(simiOpType)
		}
		return lastLex
	}
}

// lexLabel will tokenize a javascript label, starting after the label
func lexLabel(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		switch {
		case lex.acceptKeyword(ifKeyword):
			lex.emit(keywordType)
			return lexIfStmt(lastLex)
		case lex.acceptKeyword(forKeyword), lex.acceptKeyword(whileKeyword), lex.acceptKeyword(switchKeyword), lex.acceptKeyword(switchKeyword), lex.acceptKeyword(withKeyword):
			lex.emit(keywordType)
			return lexCtrlStruct(lastLex)
		case lex.acceptExact(doKeyword):
			lex.emit(keywordType)
			return lexDoWhile(lastLex)
		case lex.acceptExact(tryKeyword):
			lex.emit(keywordType)
			return lexTryCatch(lastLex)
		case lex.acceptExact(classKeyword):
			lex.emit(keywordType)
			return lexClass(lastLex)
		case lex.acceptExact(curlyOpen):
			lex.skip(newCurlyGroupSkipper(), nil)
			lex.emit(codeBlockType)
			return lastLex
		case lex.acceptVarName():
			lex.emit(varNameType)

			if !lex.acceptExact(eqOp) {
				lex.emitError("Variable name that is not being assigned following label")
				return nil
			}
			lex.emit(eqOpType)

			lex.acceptCodeBlock()
			lex.emit(codeBlockType)

			return lastLex
		default:
			lex.emitError("Un labelable statment following label")
			return nil
		}
	}
}

// lexFunction will tokenize a javascript function defintion, starting after the keyword
func lexFunction(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptVarName(); found {
			lex.emit(varNameType)
		}

		if found := lex.acceptExact(parenOpen); !found {
			lex.emitError("No arguments given for function")
		}
		lex.skip(newParenGroupSkipper(), nil)
		lex.emit(paramsType) //TODO, add this type

		if found := lex.acceptExact(curlyOpen); !found {
			lex.emitError("No body given for function")
		}
		lex.skip(newCurlyGroupSkipper(), nil)
		lex.emit(codeBlockType) //TODO, add this type (maybe replace expr)
		return lastLex
	}
}

// lexIfStmt will tokenize a javascript if/else control structures, starting after the keyword
func lexIfStmt(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptExact(parenOpen); !found {
			lex.emitError("No params given for if stmt")
		}
		lex.skip(newParenGroupSkipper(), nil)
		lex.emit(paramsType)

		if found := lex.acceptExact(curlyOpen); found {
			lex.skip(newCurlyGroupSkipper(), nil)
			lex.emit(codeBlockType)
		} else {
			lex.acceptCodeBlock()
			lex.emit(codeBlockType)
			if lex.acceptExact(simiOp) {
				lex.emit(simiOpType)
			}
		}

		for {
			if found := lex.acceptExact(elseKeyword); !found {
				//TODO, add this type (replace other keywords)
				return lastLex
			}
			lex.emit(keywordType)

			if found := lex.acceptExact(ifKeyword); found {
				lex.emit(keywordType)
				if found := lex.acceptExact(parenOpen); !found {
					lex.emitError("No params given for else if stmt")
				}
				lex.skip(newParenGroupSkipper(), nil)
				lex.emit(paramsType)
			}

			if found := lex.acceptExact(curlyOpen); found {
				lex.skip(newCurlyGroupSkipper(), nil)
				lex.emit(codeBlockType)
			} else {
				lex.acceptCodeBlock()
				lex.emit(codeBlockType)
				if lex.acceptExact(simiOp) {
					lex.emit(simiOpType)
				}
			}
		}
	}
}

// lexCtrlStruct will tokenize a javascript while/for/switch/with control structures, starting after the keyword
func lexCtrlStruct(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptExact(parenOpen); !found {
			lex.emitError("No params given for control structure")
		}
		lex.skip(newParenGroupSkipper(), nil)
		lex.emit(paramsType)
		lex.acceptSpaces()

		if found := lex.acceptExact(curlyOpen); found {
			lex.skip(newCurlyGroupSkipper(), nil)
			lex.emit(codeBlockType)
		} else {
			lex.acceptCodeBlock()
			lex.emit(codeBlockType)
			if lex.acceptExact(simiOp) {
				lex.emit(simiOpType)
			}
		}

		return lastLex
	}
}

// lexDoWhile will tokenize a javascript do while control structures, starting after the keyword
func lexDoWhile(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptExact(curlyOpen); found {
			lex.skip(newCurlyGroupSkipper(), nil)
			lex.emit(codeBlockType)
		} else {
			lex.acceptCodeBlock()
			lex.emit(codeBlockType)
			if lex.acceptExact(simiOp) {
				lex.emit(simiOpType)
			}
		}

		if found := lex.acceptKeyword(whileKeyword); !found {
			lex.emitError("Do while loop has no while condition")
			return nil
		}
		lex.emit(keywordType)

		if found := lex.acceptExact(parenOpen); !found {
			lex.emitError("No params given for do while loop")
			return nil
		}
		lex.skip(newParenGroupSkipper(), nil)
		lex.emit(paramsType)

		if found := lex.acceptExact(simiOp); !found {
			lex.emitError("No semicolon after do while loop")
			return nil
		}
		lex.emit(simiOpType)

		return lastLex
	}
}

// lexTryCatch will tokenize a javascript try catch control structures, starting after the keyword
func lexTryCatch(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptExact(curlyOpen); !found {
			lex.emitError("No body given for try")
			return nil
		}
		lex.skip(newCurlyGroupSkipper(), nil)
		lex.emit(codeBlockType)

		if found := lex.acceptKeyword(catchKeyword); !found {
			lex.emitError("No catch given for try")
			return nil
		}
		lex.emit(keywordType)

		if found := lex.acceptExact(parenOpen); found {
			lex.skip(newParenGroupSkipper(), nil)
			lex.emit(paramsType)
		}

		if found := lex.acceptExact(curlyOpen); !found {
			lex.emitError("No body given for catch")
			return nil
		}
		lex.skip(newCurlyGroupSkipper(), nil)
		lex.emit(codeBlockType)

		if found := lex.acceptKeyword(finallyKeyword); !found {
			return lastLex
		}
		lex.emit(keywordType)

		if found := lex.acceptExact(curlyOpen); !found {
			lex.emitError("No body given for finally")
			return nil
		}
		lex.skip(newCurlyGroupSkipper(), nil)
		lex.emit(codeBlockType)

		return lastLex
	}
}

// lexClass will tokenize a javascript class, starting after the keyword
func lexClass(lastLex lexFn) lexFn {
	return func(lex *codeLexer) lexFn {
		if found := lex.acceptVarName(); found {
			lex.emit(varNameType)
		}

		if found := lex.acceptKeyword(extendsKeyword); found {
			lex.emit(keywordType)

			if found := lex.acceptVarName(); !found {
				lex.emitError("No super class given")
			}
			lex.emit(varNameType)
		}

		if found := lex.acceptExact(curlyOpen); !found {
			lex.emitError("No body given for class")
			return nil
		}
		lex.skip(newCurlyGroupSkipper(), nil)
		lex.emit(codeBlockType)

		return lastLex
	}
}

// The noCommentLexer parses and stores comments instead of emitting them.
type noCommentLexer struct {
	lex      *lexer
	comments *childComments
}

func (lex *noCommentLexer) Next() (tokenType, []byte) {
	tt, data := lex.lex.Next()
	if tt == commentType {
		lex.lex.rewind(tt, data)
		node := &CommentNode{}
		node.parse(lex.lex)
		lex.comments.appendChild(node)

		tt, data = lex.lex.Next()
	} else {
		lex.comments.appendNil()
	}

	return tt, data
}

func (lex *noCommentLexer) rewind(tt tokenType, data []byte) {
	lex.comments.pop()
	lex.lex.rewind(tt, data)
}
