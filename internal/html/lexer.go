package html

import "github.com/tdewolff/parse/v2/html"

// All lexers types for html will implment the Lexer interface.
type Lexer interface {
	Next() (html.TokenType, []byte)
	Err() error
}

type wrappedLexer struct {
	tt        html.TokenType
	data      []byte
	canUnwrap bool
	l         Lexer
}

func wrapLexer(tt html.TokenType, data []byte, l Lexer) *wrappedLexer {
	if wl, ok := l.(*wrappedLexer); ok {
		return &wrappedLexer{tt, data, false, wl.unwrap()}
	}

	return &wrappedLexer{tt, data, false, l}
}

func (l *wrappedLexer) unwrap() Lexer {
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
