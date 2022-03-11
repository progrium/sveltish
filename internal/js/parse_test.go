package js

import (
	"bytes"
	"testing"
)

func TestParseAndReprint(t *testing.T) {
	testData := []struct {
		name  string
		input []byte
	}{
		{"SingleComment", []byte("// Some Comment")},
		{"MutlipleComments", []byte(
			`// Some Comment
			/*
				Another Comment
			*/`,
		)},
		{"VariableDeclaration", []byte("let some = 'value';")},
		{"FunctionDeclaration", []byte(
			`function func(args) {
				return 'value';
			}`,
		)},
		{"ClassDeclaration", []byte(
			`class SomeClass {
				method() {
					return 'value';
				}
			}`,
		)},
		{"MultipleDeclarations", []byte(
			`let some = 'value';

			// Some Comment
			function Func(args) {
				return 'value';
			}

			class SomeClass {
				method() {
					return 'value';
				}
			}`,
		)},
		{"IfNoElseStatement", []byte(
			`if (test = 'value') {
				func()
			}`,
		)},
		{"IfElseStatement", []byte(
			`if (test = 'value') {
				func()
			} else {
				anotherFunc()
			}`,
		)},
		{"IfElseIfStatement", []byte(
			`if (test = 'value') {
				func()
			} else if (test = 'diffrent value') {
				anotherFunc()
			} else {
				finalFunc()
			}`,
		)},
		{"ForLoop", []byte(
			`for (let i=0; i<100; i++) {
				func(i);
			}`,
		)},
		{"DoWhileLoop", []byte(
			`do {
				func(i);
				i++;
			} while(i<100);`,
		)},
		{
			"LabeledAssignment",
			[]byte("$: some = 'value';"),
		},
		{
			"LabeledStatment",
			[]byte(
				`$: if (test = 'value') {
					func();
				}`,
			),
		},
		{
			"LabeledBlock",
			[]byte(
				`$: {
					func();
				}`,
			),
		},
		{"MultipleStatements", []byte(
			`for (let i=0; i<100; i++) {
				func(i);
			}

			if (test = 'value') {
				func()
			} else if (test = 'diffrent value') {
				anotherFunc()
			} else {
				finalFunc()
			}`,
		)},
		{"RootExpr", []byte(
			`some.func({
				param: 'name',
				method() {
					reutnr 'value';
				}
			});`,
		)},
	}

	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			script, err := Parse(bytes.NewReader(td.input))
			if err != nil {
				t.Fatalf("Parse return error: %q", err.Error())
			}

			if js := script.Js(); js != string(td.input) {
				t.Fatalf("Parsed %q but output %q", td.input, js)
			}
		})
	}
}

func TestIsFunc(t *testing.T) {
	testData := []struct {
		name   string
		input  []byte
		output bool
	}{
		{
			"NamedFunc",
			[]byte(`function func() {
				do("somthing");
			}`),
			true,
		},
		{
			"NamedFuncWithArgs",
			[]byte(`function func(a, b) {
				do("somthing");
			}`),
			true,
		},
		{
			"AnonymousFunc",
			[]byte("function() { do('somthing'); }"),
			true,
		},
		{
			"ArrowFunc",
			[]byte("() => do('somthing')"),
			true,
		},
		{
			"ArrowFuncWithArgs",
			[]byte("(a, b) => do('somthing')"),
			true,
		},
		{
			"NoParenArrowFunc",
			[]byte("a => do('somthing')"),
			true,
		},
		{
			"MultiLineArrowFunc",
			[]byte(`function func() {
				do("somthing");
			}`),
			true,
		},
		{
			"Object",
			[]byte("{ some: 'value' }"),
			false,
		},
		{
			"Paren",
			[]byte("(skippedValue, usedValue)"),
			false,
		},
		{
			"String",
			[]byte("'function (not)'"),
			false,
		},
	}

	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			result := IsFunc(td.input)

			if td.output != result {
				t.Fatalf("Expected %t but found %t for %q", td.output, result, td.input)
			}
		})
	}
}
