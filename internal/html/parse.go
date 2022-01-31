package html

import (
	"errors"
	"io"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/html"
)

// Parse will take the html source and create a Doc node from it.
func Parse(src io.Reader) (*Doc, error) {
	n := &Doc{}
	if _, err := n.Parse(html.NewLexer(parse.NewInput(src))); err != nil {
		return n, err
	}

	return n, nil
}

func (n *Doc) Parse(l Lexer) (Lexer, error) {
	var err error
	for err == nil {
		l, err = parseNextChild(l, n)
	}

	if err != io.EOF {
		return l, err
	}
	return l, nil
}

func (n *ElNode) Parse(l Lexer) (Lexer, error) {
	tt, data := l.Next()
	if tt != html.StartTagToken {
		return l, errors.New("Invalid parser position passed to elNode.Parse")
	}
	n.Tag = string(data[1:])

	var err error
	l, err = parseAttr(l, n)
	if err != nil {
		return l, err
	}

	tt, data = l.Next()
	for tt != html.EndTagToken {
		nextL, err := parseNextChild(wrapLexer(tt, data, l), n)
		if err != nil {
			return nextL, err
		}

		l = nextL
		tt, data = l.Next()
	}
	return l, nil
}

func (n *LeafElNode) Parse(l Lexer) (Lexer, error) {
	tt, data := l.Next()
	switch tt {
	case html.SvgToken:
		return l, errors.New("NYI: parsing <svg />")
	case html.MathToken:
		return l, errors.New("NYI: parsing <math />")
	case html.StartTagToken:
		n.Tag = string(data[1:])
	default:
		return l, errors.New("Invalid parser position passed to leafElNode.Parse")
	}

	var err error
	l, err = parseAttr(l, n)
	if err != nil {
		return l, err
	}

	tt, data = l.Next()
	if tt == html.TextToken {
		n.Content = string(data)
		tt, data = l.Next()
	}
	if tt != html.EndTagToken {
		return l, errors.New("Invalid parser token in leaf element")
	}

	return l, nil
}

func (n *TxtNode) Parse(l Lexer) (Lexer, error) {
	tt, data := l.Next()
	if tt != html.TextToken {
		return l, errors.New("Invalid parser position passed to txtNode.Parse")
	}

	//TODO, parse {...} for content
	n.Content = string(data)
	return l, nil
}

func parseNextChild(l Lexer, n NodeContainer) (Lexer, error) {
	var err error

	tt, data := l.Next()
	switch tt {
	case html.StartTagToken:
		var newNode Node
		tag := string(data[1:])
		if tag == "script" || tag == "style" {
			newNode = &LeafElNode{}
		} else {
			newNode = &ElNode{}
		}

		l, err = newNode.Parse(wrapLexer(tt, data, l))
		if err != nil {
			return l, err
		}

		n.AppendChild(newNode)
		return l, nil
	case html.SvgToken:
	case html.MathToken:
		newNode := &LeafElNode{}
		l, err = newNode.Parse(wrapLexer(tt, data, l))
		if err != nil {
			return l, err
		}

		n.AppendChild(newNode)
		return l, nil
	case html.TextToken:
		newNode := &TxtNode{}
		l, err = newNode.Parse(wrapLexer(tt, data, l))
		if err != nil {
			return l, err
		}

		n.AppendChild(newNode)
		return l, nil
	case html.CommentToken:
		return l, nil
	case html.ErrorToken:
		return l, l.Err()
	}

	return l, errors.New("invalid token in children")
}

func parseAttr(l Lexer, n Node) (Lexer, error) {
	tt, _ := l.Next()
	for tt != html.StartTagCloseToken {
		if tt != html.AttributeToken {
			return l, errors.New("Invalid token when attribute expected")
		}

		//TODO, parse attributes
		tt, _ = l.Next()
	}
	return l, nil
}
