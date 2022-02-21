package js

import (
	"testing"
)

func TestSkipGroup(t *testing.T) {
	testData := []struct {
		name  string
		input []byte
	}{
		{"VariableName", []byte("{varName}")},
		{"VariableWithComment", []byte("{varName/* Some comment with }*/}")},
		{"BooleanExpression", []byte("{varName === 100}")},
		{"ArrayLookup", []byte("{varName[100]}")},
		{"ObjectValue", []byte("{varName.value}")},
		{"FuctionCall", []byte("{func(100)}")},
		{"MethodCall", []byte("{varName.func(100)}")},
		{"RegexString", []byte(`{/ab+c/}`)},
		{"FuctionCallWithString", []byte("{func('test string')}")},
		{"FuctionCallWithComplexString", []byte("{func('test string with } and `')}")},
		{"FuctionCallWithTemplateString", []byte("{func(`test string with ${innerValue} and '`)}")},
		{"FuctionCallWithStringWithEscapedQuote", []byte(`{func('test string with \' and \\\'')}`)},
		{"FuctionCallWithStringWithBackslash", []byte(`{func('test string with \\')}`)},
		{"FuctionCallWithObject", []byte("{func({ value: '100' })}")},
		{"FuctionCallWithCallback", []byte("{func(() => { return 'value'; })}")},
		{"FuctionCallWithCallbackReturningObject", []byte("{func(() => ({ some: 'value' }))}")},
		{
			"FuctionCallWithComplexCallback",
			[]byte(`{func(() => {
				// Some comment with 
				const value = "test string with ) and }";

				return '"' + value + '"';
			})}`),
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			skpr := newCurlyGroupSkipper()
			i := 1
			for skpr.isOpen() {
				if i >= len(td.input) {
					t.Fatalf("Group still open (with %d left) after end of input", skpr.count)
				}

				skpr.next(td.input[i])
				i += 1
			}

			if i != len(td.input) {
				t.Fatalf("Closed group at %d instead of %d", i, len(td.input)-1)
			}
		})
	}
}

func TestSkipQuote(t *testing.T) {
	testData := []struct {
		name  string
		input []byte
	}{
		{"SingleQuote", []byte("'some text'")},
		{"DoubleQuote", []byte(`"some text"`)},
		{"TmplQuote", []byte("`some text`")},
		{"RegexQuote", []byte(`/ab+c/`)},
		{"QuoteWithEscape", []byte(`'some text with \''`)},
		{"QuoteWithMultiEscape", []byte(`'some text with \\'`)},
		{"TmplWithExpr", []byte("`some ${more} text`")},
		{"TmplWithTmplInExpr", []byte("`some ${func(() => `another ${tmpl}`)} text`")},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			skpr := newQuoteSkipper(td.input[0])
			i := 1
			for skpr.isOpen() {
				if i >= len(td.input) {
					t.Fatal("Quote still open after end of input")
				}

				skpr.next(td.input[i])
				i += 1
			}

			if i != len(td.input) {
				t.Fatalf("Closed quote at %d instead of %d", i, len(td.input))
			}
		})
	}
}

func TestSkipComment(t *testing.T) {
	testData := []struct {
		name  string
		open  string
		close string
		input []byte
	}{
		{"OneLineComment", lineCommentOpen, newLine, []byte("//Some Comment\n")},
		{"BlockComment", blockCommentOpen, blockCommentClose, []byte("/* Some Comment */")},
		{
			"MultiLineBlockComment",
			blockCommentOpen,
			blockCommentClose,
			[]byte(
				`/*
	Some Comment
*/`,
			),
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			skpr := newCommentSkipper([]byte(td.open), []byte(td.close))
			i := 1
			for skpr.isOpen() {
				if i >= len(td.input) {
					t.Fatal("Comment still open after end of input")
				}

				skpr.next(td.input[i])
				i += 1
			}

			if i != len(td.input) {
				t.Fatalf("Closed comment at %d instead of %d", i, len(td.input))
			}
		})
	}
}
