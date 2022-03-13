package js

import (
	"errors"
	"io"
)

// Parse will take the js source and create a Script node from it.
func Parse(src io.Reader) (*Script, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	lex := startNewLexer(lexScript, data)

	script := &Script{}
	if err := script.parse(lex); err != nil {
		return script, err
	}

	return script, nil
}

type parser interface {
	parse(*lexer) error
}

func (n *Script) parse(lex *lexer) error {
	for tt, data := lex.Next(); tt != eofType; {
		var nextNode interface {
			Node
			parser
		}
		switch tt {
		case errorType:
			return errors.New(string(data))
		case commentType:
			nextNode = &CommentNode{}
		case labelType:
			nextNode = &LabelNode{}
		case keywordType:
			nextNode, _ = nodeForKeyword(trimLeftSpaces(data))
		}
		if nextNode == nil {
			nextNode = &BlockNode{}
		}

		lex.rewind(tt, data)
		if err := nextNode.parse(lex); err != nil {
			return err
		}
		n.appendChild(nextNode)

		tt, data = lex.Next()
	}

	return nil
}

func (n *CommentNode) parse(lex *lexer) error {
	_, data := lex.Next()
	n.content = data
	return nil
}

func (n *LabelNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	_, data := ncLex.Next()
	n.label = data

	tt, data := ncLex.Next()
	switch tt {
	case varNameType:
		n.name = data

		tt, data = ncLex.Next()
		if tt != eqOpType {
			return errors.New("Equal sign must follow variable name in labeled assignment")
		}
		n.equals = data

		tt, data = ncLex.Next()
		if tt != codeBlockType {
			return errors.New("Expression must follow equals sign in labled assigment")
		}
		nextNode := &BlockNode{}
		ncLex.rewind(tt, data)
		if err := nextNode.parse(lex); err != nil {
			return err
		}
		n.body = nextNode

		tt, data = ncLex.Next()
		if tt == simiOpType {
			n.simi = data
		} else {
			ncLex.rewind(tt, data)
		}
	case keywordType:
		nextNode, ok := nodeForKeyword(trimLeftSpaces(data))
		if !ok {
			return errors.New("Invalid keyword following label")
		}
		ncLex.rewind(tt, data)
		if err := nextNode.parse(lex); err != nil {
			return err
		}
		n.body = nextNode
	case codeBlockType:
		nextNode := &BlockNode{}
		ncLex.rewind(tt, data)
		if err := nextNode.parse(lex); err != nil {
			return err
		}
		n.body = nextNode
	}
	return nil
}

func (n *VarNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.keyword = data

	tt, data = ncLex.Next()
	if tt != varNameType {
		return errors.New("Variable not given a name")
	}
	n.name = data

	tt, data = ncLex.Next()
	if tt == simiOpType {
		n.simi = data
		return nil
	}
	if tt != eqOpType {
		ncLex.rewind(tt, data)
		return nil
	}
	n.equals = data

	tt, data = ncLex.Next()
	if tt != codeBlockType {
		return errors.New("Variable with '=' sign not given a value")
	}
	n.value = &BlockNode{}
	lex.rewind(tt, data)
	if err := n.value.parse(lex); err != nil {
		return err
	}

	tt, data = ncLex.Next()
	if tt == simiOpType {
		n.simi = data
	} else {
		ncLex.rewind(tt, data)
	}
	return nil
}

func (n *FuncNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.keyword = data

	tt, data = ncLex.Next()
	if tt == varNameType {
		n.name = data

		tt, data = ncLex.Next()
	}
	if tt != paramsType {
		return errors.New("Function not given arguments")
	}
	n.params = data

	tt, data = ncLex.Next()
	if tt != codeBlockType {
		return errors.New("Function not given body")
	}
	n.body = &BlockNode{}
	lex.rewind(tt, data)
	return n.body.parse(lex)
}

func (n *ClassNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.classKeyword = data

	tt, data = ncLex.Next()
	if tt == varNameType {
		n.name = data

		tt, data = ncLex.Next()
	}
	if tt == keywordType {
		if trimLeftSpaces(data) != extendsKeyword {
			return errors.New("Invalid keyword in class declaration")
		}
		n.extendsKeyword = data

		tt, data = ncLex.Next()
		if tt == varNameType {
			return errors.New("No super class name after extends keyword")
		}
		n.superName = data

		tt, data = ncLex.Next()
	}

	if tt != codeBlockType {
		return errors.New("Class not given a body")
	}
	n.body = &BlockNode{}
	lex.rewind(tt, data)
	return n.body.parse(lex)
}

func (n *IfNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.ifKeyword = data

	tt, data = ncLex.Next()
	if tt != paramsType {
		return errors.New("If statement not given parameters")
	}
	n.params = data

	tt, data = ncLex.Next()
	if tt != codeBlockType {
		return errors.New("If statement not given body")
	}
	ncLex.rewind(tt, data)
	n.ifBody = &BlockNode{}
	if err := n.ifBody.parse(lex); err != nil {
		return err
	}

	tt, data = ncLex.Next()
	if tt != keywordType || trimLeftSpaces(data) != elseKeyword {
		ncLex.rewind(tt, data)
		return nil
	}
	n.elseKeyword = data

	tt, data = ncLex.Next()
	if tt == codeBlockType {
		ncLex.rewind(tt, data)
		n.elseBody = &BlockNode{}
		return n.elseBody.parse(lex)
	}
	if tt != keywordType || trimLeftSpaces(data) != ifKeyword {
		return errors.New("Invalid input following else keyword")
	}

	ncLex.rewind(tt, data)
	n.elseIfNode = &IfNode{}
	return n.elseIfNode.parse(lex)
}

func (n *basicCtrlStructNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.keyword = data

	tt, data = ncLex.Next()
	if tt != paramsType {
		return errors.New("Control structure not given parameters")
	}
	n.params = data

	tt, data = ncLex.Next()
	if tt != codeBlockType {
		return errors.New("Control structure not given body")
	}
	n.body = &BlockNode{}
	lex.rewind(tt, data)
	return n.body.parse(lex)
}

func (n *DoWhileLoopNode) parse(lex *lexer) error {
	n.comments = &childComments{}
	ncLex := &noCommentLexer{lex, n.comments}

	tt, data := ncLex.Next()
	n.doKeyword = data

	tt, data = ncLex.Next()
	if tt != codeBlockType {
		return errors.New("Do while loop not given body")
	}
	n.body = &BlockNode{}
	lex.rewind(tt, data)
	if err := n.body.parse(lex); err != nil {
		return err
	}

	tt, data = ncLex.Next()
	if tt != keywordType || trimLeftSpaces(data) != whileKeyword {
		return errors.New("Do while loop expected while keyword after body")
	}
	n.whileKeyword = data

	tt, data = ncLex.Next()
	if tt != paramsType {
		return errors.New("Do while loop not given parameters")
	}
	n.params = data

	tt, data = ncLex.Next()
	if tt == simiOpType {
		n.simi = data
	} else {
		ncLex.rewind(tt, data)
	}
	return nil
}

func (n *BlockNode) parse(lex *lexer) error {
	_, data := lex.Next()
	n.content = data
	return nil
}

func nodeForKeyword(kw string) (interface {
	Node
	parser
}, bool) {
	switch kw {
	case varKeyword, letKeyword, constKeyword:
		return &VarNode{}, true
	case funcKeyword:
		return &FuncNode{}, true
	case classKeyword:
		return &ClassNode{}, true
	case ifKeyword:
		return &IfNode{}, true
	case switchKeyword:
		return &SwitchNode{}, true
	case withKeyword:
		return &WithNode{}, true
	case forKeyword:
		return &ForLoopNode{}, true
	case whileKeyword:
		return &WhileLoopNode{}, true
	case doKeyword:
		return &DoWhileLoopNode{}, true
	}

	return nil, false
}

// IsFunc parses the given data and checks if it is prefixed like a function
// call.
func IsFunc(data []byte) bool {
	lex := newCodeLexer(data)
	defer close(lex.items) //NOTE, .items not used so probably not needed

	lex.acceptSpaces()
	switch {
	case lex.acceptKeyword(funcKeyword):
		return true
	case lex.acceptExact(parenOpen):
		lex.skip(newParenGroupSkipper(), nil)
		lex.acceptSpaces()
		return lex.acceptExact(arrowFuncOp)
	case lex.acceptVarName():
		lex.acceptSpaces()
		return lex.acceptExact(arrowFuncOp)
	}
	return false
}
