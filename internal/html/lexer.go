package html

import (
	"io"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/html"
)

type lexer struct {
	lex   *html.Lexer
	stack []lexerOutput
}

type lexerOutput struct {
	tt   html.TokenType
	data []byte
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		html.NewLexer(parse.NewInput(src)),
		[]lexerOutput{},
	}
}

func (lex *lexer) rewind(tt html.TokenType, data []byte) {
	lex.stack = append(lex.stack, lexerOutput{tt, data})
}

func (lex *lexer) Next() (html.TokenType, []byte) {
	stackSize := len(lex.stack)
	if stackSize == 0 {
		return lex.lex.Next()
	}

	info := lex.stack[stackSize-1]
	lex.stack = lex.stack[:stackSize-1]

	return info.tt, info.data
}

func (lex *lexer) Err() error {
	return lex.lex.Err()
}

func isWhiteSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}
