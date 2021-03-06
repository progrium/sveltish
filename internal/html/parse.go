package html

import (
	"errors"
	"io"

	"github.com/tdewolff/parse/v2/html"
)

type parser interface {
	parse(*idGenerator, *lexer) error
}

type idGenerator struct {
	id NodeId
}

func (idg *idGenerator) next() NodeId {
	id := idg.id
	idg.id += 1
	return id
}

// Parse will take the html source and create a Doc node from it.
func Parse(src io.Reader) (*Doc, error) {
	doc := &Doc{}
	if err := doc.parse(&idGenerator{}, newLexer(src)); err != nil {
		return doc, err
	}

	return doc, nil
}

func (n *Doc) parse(idg *idGenerator, lex *lexer) error {
	n.id = idg.next()

	var err error
	for err == nil {
		err = parseNextChild(n, idg, lex)
	}
	if err != io.EOF {
		return err
	}

	return nil
}

func (n *ElNode) parse(idg *idGenerator, lex *lexer) error {
	n.id = idg.next()

	tt, data := lex.Next()
	if tt != html.StartTagToken {
		return errors.New("Invalid parser position passed to elNode.parse")
	}
	n.tag = string(data[1:])

	if err := parseAttr(n, lex); err != nil {
		return err
	}

	tt, data = lex.Next()
	for tt != html.EndTagToken {
		lex.rewind(tt, data)
		if err := parseNextChild(n, idg, lex); err != nil {
			return err
		}

		tt, data = lex.Next()
	}
	return nil
}

func (n *LeafElNode) parse(idg *idGenerator, lex *lexer) error {
	n.id = idg.next()

	tt, data := lex.Next()
	switch tt {
	case html.SvgToken:
		return errors.New("NYI: parsing <svg />")
	case html.MathToken:
		return errors.New("NYI: parsing <math />")
	case html.StartTagToken:
		n.tag = string(data[1:])
	default:
		return errors.New("Invalid parser position passed to leafElNode.parse")
	}

	err := parseAttr(n, lex)
	if err != nil {
		return err
	}

	tt, data = lex.Next()
	if tt == html.TextToken {
		n.content = string(data)
		tt, data = lex.Next()
	}
	if tt != html.EndTagToken {
		return errors.New("Invalid parser token in leaf element")
	}

	return nil
}

func (n *TxtNode) parse(idg *idGenerator, lex *lexer) error {
	n.id = idg.next()

	tt, data := lex.Next()
	if tt != html.TextToken {
		return errors.New("Invalid parser position passed to txtNode.parse")
	}

	exprIndex := indexStartExpr(data)
	if exprIndex != -1 {
		lex.rewind(tt, data[exprIndex:])
		data = data[:exprIndex]
	}

	n.content = string(data)
	return nil
}

func (n *ExprNode) parse(idg *idGenerator, lex *lexer) error {
	n.id = idg.next()

	tt, data := lex.Next()
	if tt != html.TextToken || data[0] != '{' {
		return errors.New("Invalid parser position passed to DyncTxtNode.parse")
	}

	txtIndex := indexAfterExpr(data)
	if txtIndex == -1 {
		return errors.New("Expression was opened but not closed")
	}
	if txtIndex != len(data) {
		lex.rewind(tt, data[txtIndex:])
		data = data[:txtIndex]
	}

	n.js = string(data[1 : len(data)-1])
	return nil
}

func parseNextChild(n mutableContainer, idg *idGenerator, lex *lexer) error {
	tt, data := lex.Next()
	switch tt {
	case html.StartTagToken:
		var newNode interface {
			Node
			parser
		}

		switch string(data) {
		case "<script", "<style":
			newNode = &LeafElNode{}
		default:
			newNode = &ElNode{}
		}

		lex.rewind(tt, data)
		if err := newNode.parse(idg, lex); err != nil {
			return err
		}

		n.appendChild(newNode)
		return nil
	case html.SvgToken:
	case html.MathToken:
		newNode := &LeafElNode{}

		lex.rewind(tt, data)
		if err := newNode.parse(idg, lex); err != nil {
			return err
		}

		n.appendChild(newNode)
		return nil
	case html.TextToken:
		var newNode interface {
			Node
			parser
		}
		if data[0] == '{' {
			newNode = &ExprNode{}
		} else {
			newNode = &TxtNode{}
		}

		lex.rewind(tt, data)
		if err := newNode.parse(idg, lex); err != nil {
			return err
		}

		n.appendChild(newNode)
		return nil
	case html.CommentToken:
		return nil
	case html.ErrorToken:
		return lex.Err()
	}

	return errors.New("invalid token in children")
}

func parseAttr(n mutableElement, lex *lexer) error {
	tt, data := lex.Next()
	for tt != html.StartTagCloseToken {
		if tt != html.AttributeToken {
			return errors.New("Invalid token when attribute expected")
		}

		attr, err := newAttr(data)
		if err != nil {
			return err
		}
		n.appendAttr(attr)

		tt, data = lex.Next()
	}
	return nil
}
