package html

import (
	"bytes"
	"testing"
)

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
