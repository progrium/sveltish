package html

import (
	"testing"

	"github.com/progrium/sveltish/internal/js"
)

func TestNewAttr(t *testing.T) {
	testData := []struct {
		name          string
		input         []byte
		attrType      string
		attrName      string
		attrDir       string
		attrJsContent string //TODO, replace for Rewriter
	}{
		{
			"NameOnly",
			[]byte("value"),
			"static",
			"value",
			"",
			"''",
		},
		{
			"NameWithHyphenOnly",
			[]byte("data-value"),
			"static",
			"data-value",
			"",
			"''",
		},
		{
			"NameWithWhitespace",
			[]byte(" value"),
			"static",
			"value",
			"",
			"''",
		},
		{
			"EmptyStringValue",
			[]byte(`value=""`),
			"static",
			"value",
			"",
			"''",
		},
		{
			"StringValue",
			[]byte(`value="test value"`),
			"static",
			"value",
			"",
			"'test value'",
		},
		{
			"StringWithQuoteValue",
			[]byte(`value="test'ng value"`),
			"static",
			"value",
			"",
			"'test\\'ng value'",
		},
		{
			"StringWithHyphenInName",
			[]byte(`data-value="test value"`),
			"static",
			"data-value",
			"",
			"'test value'",
		},
		{
			"StringWithWhiteSpace",
			[]byte(` value="test value"`),
			"static",
			"value",
			"",
			"'test value'",
		},
		{
			"StringWithOnlyDir",
			[]byte("class:name"),
			"static",
			"class",
			"name",
			"''",
		},
		{
			"StringWithDirAndVal",
			[]byte(`bind:innerText="Some Text"`),
			"static",
			"bind",
			"innerText",
			"'Some Text'",
		},
		{
			"ExprWithJustAnExpr",
			[]byte(`value="{testValue}"`),
			"expr",
			"value",
			"",
			"testValue",
		},
		{
			"ExprWithJustAnComplexExpr",
			[]byte(`value="{testValue === ` + "`some ${value}`" + `}"`),
			"expr",
			"value",
			"",
			"testValue === `some ${value}`",
		},
		{
			"ExprWithHyphenInName",
			[]byte(`data-value="{testValue}"`),
			"expr",
			"data-value",
			"",
			"testValue",
		},
		{
			"ExprWithWhitespace",
			[]byte(` value="{testValue}"`),
			"expr",
			"value",
			"",
			"testValue",
		},
		{
			"ExprWithDirAndVal",
			[]byte(`on:click="{handleClick}"`),
			"expr",
			"on",
			"click",
			"handleClick",
		},
		{
			"TmplWithOneExprAtStart",
			[]byte(`value="{test} value"`),
			"tmpl",
			"value",
			"",
			"`${test} value`",
		},
		{
			"TmplWithOneExprAtEnd",
			[]byte(`value="test {value}"`),
			"tmpl",
			"value",
			"",
			"`test ${value}`",
		},
		{
			"TmplWithOneExprInMiddle",
			[]byte(`value="some {test} value"`),
			"tmpl",
			"value",
			"",
			"`some ${test} value`",
		},
		{
			"TmplWithHyphen",
			[]byte(`data-value="test {value}"`),
			"tmpl",
			"data-value",
			"",
			"`test ${value}`",
		},
		{
			"TmplWithWhitespace",
			[]byte(` value="test {value}"`),
			"tmpl",
			"value",
			"",
			"`test ${value}`",
		},
		{
			"TmplWithComplexExpr",
			[]byte(`value="a {testValue === ` + "`some ${value}`" + `} value"`),
			"tmpl",
			"value",
			"",
			"`a ${testValue === `some ${value}`} value`",
		},
		{
			"TmplWithMultipleExprs",
			[]byte(`value="some {test} {value}"`),
			"tmpl",
			"value",
			"",
			"`some ${test} ${value}`",
		},
		{
			"TmplWithDirAndVal",
			[]byte(`bind:innerText="Some {text}"`),
			"tmpl",
			"bind",
			"innerText",
			"`Some ${text}`",
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			attr, _ := newAttr(td.input)

			switch attr.(type) {
			case *staticAttr:
				if td.attrType != "static" {
					t.Fatalf("Attr should be %q, but a staticAttr was created", td.attrType)
				}
			case *exprAttr:
				if td.attrType != "expr" {
					t.Fatalf("Attr should be %q, but a exprAttr was created", td.attrType)
				}
			case *tmplAttr:
				if td.attrType != "tmpl" {
					t.Fatalf("Attr should be %q, but a tmplAttr was created", td.attrType)
				}
			}

			if attr.Name() != td.attrName {
				t.Fatalf("Attr should have name %q but it is %q", td.attrName, attr.Name())
			}

			dir, exists := attr.Dir()
			if td.attrDir == "" && exists {
				t.Fatalf("Attr should not have a directive, but one exists with value %q", dir)
			}
			if td.attrDir != "" && !exists {
				t.Fatalf("Attr should have a directive, but exists check is false")
			}
			if td.attrDir != "" && dir != td.attrDir {
				t.Fatalf("Attr directive should be %q but it is %q", td.attrDir, dir)
			}

			if jsContent, _ := attr.RewriteJs(&doNothingRw{}); string(jsContent) != td.attrJsContent {
				t.Fatalf("Attr should have javascript content %q but it is %q", td.attrJsContent, jsContent)
			}
		})
	}
}

type doNothingRw struct{}

func (_ *doNothingRw) Rewrite(data []byte) ([]byte, *js.VarsInfo) {
	return data, js.NewEmptyVarsInfo()
}
