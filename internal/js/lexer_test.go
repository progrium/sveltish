package js

import (
	"bytes"
	"testing"
)

func TestScriptLexer(t *testing.T) {
	testData := []struct {
		name   string
		input  []byte
		output []lexerItem
	}{
		{
			"OneLineComment",
			[]byte("//Some Comment\n"),
			[]lexerItem{
				{commentType, []byte("//Some Comment\n")},
				{eofType, nil},
			},
		},
		{
			"CommentWithExtraNewLine",
			[]byte("//Some Comment\n\n"),
			[]lexerItem{
				{commentType, []byte("//Some Comment\n")},
				{eofType, nil},
			},
		},
		{
			"MultiLineBlockComment",
			[]byte(
				`/*
	Some Comment
*/`,
			),
			[]lexerItem{
				{
					commentType,
					[]byte(
						`/*
	Some Comment
*/`,
					),
				},
				{eofType, nil},
			},
		},
		{
			"MultipleComments",
			[]byte(
				`//Some Comment
/*
	Some Comment
*/`,
			),
			[]lexerItem{
				{commentType, []byte("//Some Comment\n")},
				{
					commentType,
					[]byte(
						`/*
	Some Comment
*/`,
					),
				},
				{eofType, nil},
			},
		},
		{
			"DeclareSingleVar",
			[]byte("var test;"),
			[]lexerItem{
				{keywordType, []byte("var")},
				{varNameType, []byte(" test")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareVarWithExtraNewLine",
			[]byte("var test;\n"),
			[]lexerItem{
				{keywordType, []byte("var")},
				{varNameType, []byte(" test")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareSingleLet",
			[]byte("let test;"),
			[]lexerItem{
				{keywordType, []byte("let")},
				{varNameType, []byte(" test")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareSingleConst",
			[]byte("const test = 'test';"),
			[]lexerItem{
				{keywordType, []byte("const")},
				{varNameType, []byte(" test")},
				{eqOpType, []byte(" =")},
				{codeBlockType, []byte(" 'test'")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareFuncCallConst",
			[]byte("const test = func(() => `some ${quote}`);"),
			[]lexerItem{
				{keywordType, []byte("const")},
				{varNameType, []byte(" test")},
				{eqOpType, []byte(" =")},
				{codeBlockType, []byte(" func(() => `some ${quote}`)")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareMethodCallConst",
			[]byte("const test = obj.method(() => `some ${quote}`);"),
			[]lexerItem{
				{keywordType, []byte("const")},
				{varNameType, []byte(" test")},
				{eqOpType, []byte(" =")},
				{codeBlockType, []byte(" obj.method(() => `some ${quote}`)")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareComplexConst",
			[]byte(`const test = func(() => {
				// Some comment
				obj = {
					method() {
						alert("Some quote")
					},
					value: 'Another quote',
				};
			});`),
			[]lexerItem{
				{keywordType, []byte("const")},
				{varNameType, []byte(" test")},
				{eqOpType, []byte(" =")},
				{
					codeBlockType,
					[]byte(
						` func(() => {
				// Some comment
				obj = {
					method() {
						alert("Some quote")
					},
					value: 'Another quote',
				};
			})`,
					),
				},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"CommentOnDeclareLet",
			[]byte(
				`//Some Comment
let test;`,
			),
			[]lexerItem{
				{commentType, []byte("//Some Comment\n")},
				{keywordType, []byte("let")},
				{varNameType, []byte(" test")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"DeclareFunction",
			[]byte("function testFunc(someArg) { return returnValue; }"),
			[]lexerItem{
				{keywordType, []byte("function")},
				{varNameType, []byte(" testFunc")},
				{paramsType, []byte("(someArg)")},
				{codeBlockType, []byte(" { return returnValue; }")},
				{eofType, nil},
			},
		},
		{
			"DeclareMutliLineFunction",
			[]byte(`function testFunc(someArg) {
				const returnValue = anotherFunc(someArg);

				// Some Comment
				return returnValue;
			}`),
			[]lexerItem{
				{keywordType, []byte("function")},
				{varNameType, []byte(" testFunc")},
				{paramsType, []byte("(someArg)")},
				{codeBlockType, []byte(
					` {
				const returnValue = anotherFunc(someArg);

				// Some Comment
				return returnValue;
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"IfStatement",
			[]byte("if (a == b) return c;"),
			[]lexerItem{
				{keywordType, []byte("if")},
				{paramsType, []byte(" (a == b)")},
				{codeBlockType, []byte(" return c")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"IfMutliLineStatement",
			[]byte(`if (a == b) {
				//Some comment
				return c;
			}`),
			[]lexerItem{
				{keywordType, []byte("if")},
				{paramsType, []byte(" (a == b)")},
				{codeBlockType, []byte(
					` {
				//Some comment
				return c;
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"IfElseStatement",
			[]byte(`if (a == b) {
				//Some comment
				return c;
			} else {
				return f({ g });
			}`),
			[]lexerItem{
				{keywordType, []byte("if")},
				{paramsType, []byte(" (a == b)")},
				{codeBlockType, []byte(
					` {
				//Some comment
				return c;
			}`,
				)},
				{keywordType, []byte(" else")},
				{codeBlockType, []byte(
					` {
				return f({ g });
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"IfElseIfStatement",
			[]byte(`if (a == b) {
				//Some comment
				return c;
			} else if (d == e) {
				return f({ g });
			} else {
				return h;
			}`),
			[]lexerItem{
				{keywordType, []byte("if")},
				{paramsType, []byte(" (a == b)")},
				{codeBlockType, []byte(
					` {
				//Some comment
				return c;
			}`,
				)},
				{keywordType, []byte(" else")},
				{keywordType, []byte(" if")},
				{paramsType, []byte(" (d == e)")},
				{codeBlockType, []byte(
					` {
				return f({ g });
			}`,
				)},
				{keywordType, []byte(" else")},
				{codeBlockType, []byte(
					` {
				return h;
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"ForLoop",
			[]byte("for (int i=0; i<a.length; i++) b[i] = a[i];"),
			[]lexerItem{
				{keywordType, []byte("for")},
				{paramsType, []byte(" (int i=0; i<a.length; i++)")},
				{codeBlockType, []byte(" b[i] = a[i]")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"ForMutliLineLoop",
			[]byte(`for (int i=0; i<a.length; i++) {
				//Some comment
				b[i] = c({ d: a[i] });
			}`),
			[]lexerItem{
				{keywordType, []byte("for")},
				{paramsType, []byte(" (int i=0; i<a.length; i++)")},
				{codeBlockType, []byte(
					` {
				//Some comment
				b[i] = c({ d: a[i] });
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"DoWhileLoop",
			[]byte(`do {
				//Some comment
				d = f({ d })
			} while (d.length != 0);`),
			[]lexerItem{
				{keywordType, []byte("do")},
				{codeBlockType, []byte(
					` {
				//Some comment
				d = f({ d })
			}`,
				)},
				{keywordType, []byte(" while")},
				{paramsType, []byte(" (d.length != 0)")},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"TryCatchStatement",
			[]byte(`try {
				//Some comment
				someFunc();
			} catch (e) {
				anotherFunc();
			}`),
			[]lexerItem{
				{keywordType, []byte("try")},
				{codeBlockType, []byte(
					` {
				//Some comment
				someFunc();
			}`,
				)},
				{keywordType, []byte(" catch")},
				{paramsType, []byte(" (e)")},
				{codeBlockType, []byte(
					` {
				anotherFunc();
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"TryCatchFinallyStatement",
			[]byte(`try {
				//Some comment
				someFunc();
			} catch (e) {
				anotherFunc();
			} finally {
				finalFunc();
			}`),
			[]lexerItem{
				{keywordType, []byte("try")},
				{codeBlockType, []byte(
					` {
				//Some comment
				someFunc();
			}`,
				)},
				{keywordType, []byte(" catch")},
				{paramsType, []byte(" (e)")},
				{codeBlockType, []byte(
					` {
				anotherFunc();
			}`,
				)},
				{keywordType, []byte(" finally")},
				{codeBlockType, []byte(
					` {
				finalFunc();
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"DeclareClass",
			[]byte(`class Name {
				constuctor(val) {
					this.val = val;
				}
				get val() {
					return this.val;
				}
				updateMethod(newVal) {
					//Some method
					this.val = newValue;
				}
			}`),
			[]lexerItem{
				{keywordType, []byte("class")},
				{varNameType, []byte(" Name")},
				{codeBlockType, []byte(
					` {
				constuctor(val) {
					this.val = val;
				}
				get val() {
					return this.val;
				}
				updateMethod(newVal) {
					//Some method
					this.val = newValue;
				}
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"DeclareSubClass",
			[]byte(`class Name extends SuperName {
				constuctor(val) {
					this.val = val;
				}
				get val() {
					return this.val;
				}
				updateMethod(newVal) {
					//Some method
					this.val = newValue;
				}
			}`),
			[]lexerItem{
				{keywordType, []byte("class")},
				{varNameType, []byte(" Name")},
				{keywordType, []byte(" extends")},
				{varNameType, []byte(" SuperName")},
				{codeBlockType, []byte(
					` {
				constuctor(val) {
					this.val = val;
				}
				get val() {
					return this.val;
				}
				updateMethod(newVal) {
					//Some method
					this.val = newValue;
				}
			}`,
				)},
				{eofType, nil},
			},
		},
		{
			"RootExpr",
			[]byte(
				`some.func({
					param: 'name',
					method() {
						return 'value';
					}
				});`,
			),
			[]lexerItem{
				{codeBlockType, []byte(
					`some.func({
					param: 'name',
					method() {
						return 'value';
					}
				})`,
				)},
				{simiOpType, []byte(";")},
				{eofType, nil},
			},
		},
		{
			"ManyStatments",
			[]byte(`
				// Some header comment
				let value = func({
					method() {
						anotherFunc();
					},
				});

				/**
				 * Some block comment.
				 */
				function func(obj) {
					obj.method();
					return something;
				}

				for (let i=0; i>value.length; i++) {
					doLoopStuff(value[0])
				}

				if (value === something) {
					doIfStuff();
				} else {
					doElseStuff();
				}
			`),
			[]lexerItem{
				{commentType, []byte("\n\t\t\t\t// Some header comment\n")},
				{keywordType, []byte("\t\t\t\tlet")},
				{varNameType, []byte(" value")},
				{eqOpType, []byte(" =")},
				{codeBlockType, []byte(
					` func({
					method() {
						anotherFunc();
					},
				})`,
				)},
				{simiOpType, []byte(";")},
				{commentType, []byte(
					`

				/**
				 * Some block comment.
				 */`,
				)},
				{keywordType, []byte("\n\t\t\t\tfunction")},
				{varNameType, []byte(" func")},
				{paramsType, []byte("(obj)")},
				{codeBlockType, []byte(
					` {
					obj.method();
					return something;
				}`,
				)},
				{keywordType, []byte("\n\n\t\t\t\tfor")},
				{paramsType, []byte(" (let i=0; i>value.length; i++)")},
				{codeBlockType, []byte(
					` {
					doLoopStuff(value[0])
				}`,
				)},
				{keywordType, []byte("\n\n\t\t\t\tif")},
				{paramsType, []byte(" (value === something)")},
				{codeBlockType, []byte(
					` {
					doIfStuff();
				}`,
				)},
				{keywordType, []byte(" else")},
				{codeBlockType, []byte(
					` {
					doElseStuff();
				}`,
				)},
				{eofType, nil},
			},
		},
	}

	for _, td := range testData {
		td := td
		t.Run(td.name, func(t *testing.T) {
			lex, _ := startNewLexer(lexScript, td.input)

			t.Logf("Testing:\n%s\n", string(td.input))
			for _, opItem := range td.output {
				tt, data := lex.Next()
				t.Logf("Item: %q with data %q", tt, data)

				if tt != opItem.tt {
					t.Fatalf("Expected token type %q but got %q", opItem.tt, tt)
				}

				if bytes.Compare(data, opItem.data) != 0 {
					t.Fatalf("Expected data %q but got %q", opItem.data, data)
				}
			}

			extra := 0
			for tt, data := lex.Next(); tt != eofType; {
				t.Logf("Extra item: %q with data %q", tt, data)

				extra += 1
			}
			if extra != 0 {
				t.Fatalf("Only expected %d lex items but found %d", len(td.output), len(td.output)+extra)
			}
		})
	}
}
