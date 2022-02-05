package html

import (
	"bytes"
	"testing"
)

func TestIndexStartExpr(t *testing.T) {
	testData := []struct {
		name  string
		input []byte
		index int
	}{
		{"NoExpression", []byte("Some text with no expression"), -1},
		{"ExpressionAtStart", []byte("{Some} text with an expression"), 0},
		{"ExpressionAtEnd", []byte("Some text with an {expression}"), 18},
		{"ExpressionInMiddle", []byte("Some text {with} an expression"), 10},
		{"NoExpressionWithEscapedBrace", []byte(`Some text with a \{ but no expression`), -1},
		{"NoExpressionWithEscapedBackslash", []byte(`Some text with a \\ but no expression`), -1},
		{"NoExpressionWithEscapedBraceAndBackslash", []byte(`Some text with a \\\{ but no expression`), -1},
		{"ExpressionBeforeEscapedBrace", []byte(`{Some} text with a \{ and an expression`), 0},
		{"ExpressionAfterEscapedBrace", []byte(`Some text with a \{ and an {expression}`), 27},
		{"ExpressionAfterEscapedBraceAndBackslash", []byte(`Some text with a \\\{ and an {expression}`), 29},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			index := indexStartExpr(td.input)
			if index != td.index {
				if index == -1 || td.index == -1 {
					t.Fatalf(
						"Expected %d from %q but found %d",
						td.index,
						td.input,
						index,
					)
				}

				t.Fatalf(
					"Expected %d from %q but found %d, spliting into %q and %q",
					td.index,
					td.input,
					index,
					td.input[:index],
					td.input[index:],
				)
			}
		})
	}
}

func TestIndexAfterExpr(t *testing.T) {
	testData := []struct {
		name  string
		input []byte
	}{
		{"VariableName", []byte("{varName}")},
		{"BooleanExpression", []byte("{varName === 100}")},
		{"ArrayLookup", []byte("{varName[100]}")},
		{"ObjectValue", []byte("{varName.value}")},
		{"FuctionCall", []byte("{func(100)}")},
		{"MethodCall", []byte("{varName.func(100)}")},
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
				const value = "test string with ) and }";

				return '"' + value + '"';
			})}`),
		},
	}
	sufixes := [][]byte{
		[]byte(""),
		[]byte(" some text after"),
		[]byte(" and {anotherExpr} plus more text"),
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			knowIndex := len(td.input)
			for _, sufix := range sufixes {
				input := append(td.input, sufix...)
				testIndex := indexAfterExpr(input)
				if testIndex == -1 {
					t.Fatalf("Returned -1 when the end exists at %d", knowIndex)
				}

				testExpr := td.input[:testIndex]
				knownExpr := td.input[:knowIndex]
				if bytes.Compare(testExpr, knownExpr) != 0 {
					t.Fatalf(
						"Found index %d insted of %d for end of expr in %q (leaving %q instead of %q)",
						testIndex,
						knowIndex,
						input,
						testExpr,
						knownExpr,
					)
				}
			}
		})
	}
}
