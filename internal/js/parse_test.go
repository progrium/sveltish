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

func TestParseAndPrintReactive(t *testing.T) {
	testData := []struct {
		name   string
		input  []byte
		output string
	}{
		{
			"LabeledAssignment",
			[]byte("$: some = 'value';"),
			`
let  some;
$$self.$$.update = () => {
$: some = 'value';
};`,
		},
		{
			"LabeledStatment",
			[]byte(
				`$: if (test = 'value') {
					func();
				}`,
			),
			`
$$self.$$.update = () => {
$: if (test = 'value') {
					func();
				}
};`,
		},
		{
			"LabeledBlock",
			[]byte(
				`$: {
					func();
				}`,
			),
			`
$$self.$$.update = () => {
$: {
					func();
				}
};`,
		},
	}
	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			script, err := Parse(bytes.NewReader(td.input))
			if err != nil {
				t.Fatalf("Parse return error: %q", err.Error())
			}

			if js := script.Js(); js != td.output {
				t.Fatalf("Expected %q but output %q", td.output, js)
			}
		})
	}
}
