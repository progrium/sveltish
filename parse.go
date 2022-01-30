package sveltish

import (
	"errors"
	"io"
	//"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/html"

	"fmt"
	"github.com/davecgh/go-spew/spew"
)

func Parse(src io.Reader, name string) (*Component, error) {
	root := &RootNode{}
	_, err := root.ParseHTML(html.NewLexer(parse.NewInput(src)))
	if err != nil {
		return nil, err
	}

	c := &Component{
		Name: name,
		JS:   &RootNode{},
		HTML: &RootNode{},
		CSS:  &RootNode{},
	}
	WalkHTML(root, func(n HTMLNode, ps []HTMLNode) bool {
		if len(ps) == 0 {
			return true
		}
		if ps[0] != root {
			panic("Parser should not walk more than the roots")
		}
		if lf, ok := n.(*LeafElNode); ok {
			switch lf.Tag {
			case "script":
				c.JS.AppendChild(lf)
				return false
			case "style":
				c.CSS.AppendChild(lf)
				return false
			}
		}

		c.HTML.AppendChild(n)
		return false
	})

	fmt.Println("-------------------")
	spew.Dump(c)
	fmt.Println("-------------------")

	return c, nil
}

type HTMLParser interface {
	ParseHTML(l HTMLLexer) (HTMLLexer, error)
}

func (n *RootNode) ParseHTML(l HTMLLexer) (HTMLLexer, error) {
	var err error
	for err == nil {
		l, err = parseNextChild(l, n)
	}

	if err != io.EOF {
		return l, err
	}
	return l, nil
}

func (n *ElNode) ParseHTML(l HTMLLexer) (HTMLLexer, error) {
	tt, data := l.Next()
	if tt != html.StartTagToken {
		return l, errors.New("Invalid parser position passed to elNode.ParseHTML")
	}
	n.Tag = string(data[1:])

	tt, data = l.Next()
	for tt != html.StartTagCloseToken {
		if tt != html.AttributeToken {
			return l, errors.New("Invalid token when attribute expected")
		}

		//TODO, parse attributes
		tt, data = l.Next()
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

func (n *LeafElNode) ParseHTML(l HTMLLexer) (HTMLLexer, error) {
	tt, data := l.Next()
	switch tt {
	case html.SvgToken:
		return l, errors.New("NYI: parsing <svg />")
	case html.MathToken:
		return l, errors.New("NYI: parsing <math />")
	case html.StartTagToken:
		n.Tag = string(data[1:])
	default:
		return l, errors.New("Invalid parser position passed to leafElNode.ParseHTML")
	}

	tt, data = l.Next()
	for tt != html.StartTagCloseToken {
		if tt != html.AttributeToken {
			return l, errors.New("Invalid token when attribute expected")
		}

		//TODO, parse attributes
		tt, data = l.Next()
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

func (n *TxtNode) ParseHTML(l HTMLLexer) (HTMLLexer, error) {
	tt, data := l.Next()
	if tt != html.TextToken {
		return l, errors.New("Invalid parser position passed to txtNode.ParseHTML")
	}

	//TODO, parse {...} for content
	n.Content = string(data)
	return l, nil
}

func parseNextChild(l HTMLLexer, n HTMLNodeContainer) (HTMLLexer, error) {
	tt, data := l.Next()
	switch tt {
	case html.StartTagToken:
		var newNode interface {
			HTMLParser
			HTMLNode
		}

		tag := string(data[1:])
		if tag == "script" || tag == "style" {
			newNode = &LeafElNode{}
		} else {
			newNode = &ElNode{}
		}

		nextL, err := newNode.ParseHTML(wrapLexer(tt, data, l))
		if err != nil {
			return nextL, err
		}

		n.AppendChild(newNode)
		return nextL, nil
	case html.SvgToken:
	case html.MathToken:
		newNode := &LeafElNode{}
		nextL, err := newNode.ParseHTML(wrapLexer(tt, data, l))
		if err != nil {
			return nextL, err
		}

		n.AppendChild(newNode)
		return nextL, nil
	case html.TextToken:
		newNode := &TxtNode{}
		nextL, err := newNode.ParseHTML(wrapLexer(tt, data, l))
		if err != nil {
			return nextL, err
		}

		n.AppendChild(newNode)
		return nextL, nil
	case html.CommentToken:
		return l, nil
	case html.ErrorToken:
		return l, l.Err()
	}

	return l, errors.New("invalid token in children")
}

type HTMLLexer interface {
	Next() (html.TokenType, []byte)
	Err() error
}

type wrappedLexer struct {
	tt        html.TokenType
	data      []byte
	canUnwrap bool
	l         HTMLLexer
}

func wrapLexer(tt html.TokenType, data []byte, l HTMLLexer) *wrappedLexer {
	if wl, ok := l.(*wrappedLexer); ok {
		return &wrappedLexer{tt, data, false, wl.unwrap()}
	}

	return &wrappedLexer{tt, data, false, l}
}

func (l *wrappedLexer) unwrap() HTMLLexer {
	if !l.canUnwrap {
		return l
	}
	if innerL, ok := l.l.(*wrappedLexer); ok {
		return innerL.unwrap()
	}

	return l.l
}

func (l *wrappedLexer) Next() (html.TokenType, []byte) {
	if l.canUnwrap {
		return l.l.Next()
	}

	l.canUnwrap = true
	return l.tt, l.data
}

func (l *wrappedLexer) Err() error {
	return l.l.Err()
}
